package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"stellarfrp/pkg/network"
	"time"
)

// NodeTrafficService 节点流量服务接口
type NodeTrafficService interface {
	RecordNodeTraffic(ctx context.Context) error
	CheckNodeStatus(ctx context.Context) error
}

// nodeTrafficService 节点流量服务实现
type nodeTrafficService struct {
	nodeRepo        repository.NodeRepository
	nodeTrafficRepo repository.NodeTrafficRepository
	logger          *logger.Logger
}

// NewNodeTrafficService 创建节点流量服务实例
func NewNodeTrafficService(
	nodeRepo repository.NodeRepository,
	nodeTrafficRepo repository.NodeTrafficRepository,
	logger *logger.Logger,
) NodeTrafficService {
	return &nodeTrafficService{
		nodeRepo:        nodeRepo,
		nodeTrafficRepo: nodeTrafficRepo,
		logger:          logger,
	}
}

// CheckNodeStatus 检查节点服务器状态并更新
func (s *nodeTrafficService) CheckNodeStatus(ctx context.Context) error {
	// 获取所有节点信息
	nodes, err := s.nodeRepo.List(ctx, 0, 10000)
	if err != nil {
		s.logger.Error("Failed to get nodes list", "error", err)
		return err
	}

	// 遍历所有节点，检查连接状态
	for _, node := range nodes {
		ip := node.IP
		port := node.FrpsPort
		status := node.Status

		// 检查端口是否可以连接
		isAvailable := network.CheckPort(ip, port)

		// 如果节点可用但状态不是work(1)，则更新为work
		if isAvailable {
			if status != 1 {
				node.Status = 1 // 在线状态
				err := s.nodeRepo.Update(ctx, node)
				if err != nil {
					s.logger.Error("Failed to update node status", "node", node.NodeName, "error", err)
					continue
				}
				s.logger.Info("Node status updated to work", "node", node.NodeName, "ip", ip, "port", port)
			}
		} else {
			// 如果节点不可用，更新为close(0)
			if status != 0 {
				node.Status = 0 // 离线状态
				err := s.nodeRepo.Update(ctx, node)
				if err != nil {
					s.logger.Error("Failed to update node status", "node", node.NodeName, "error", err)
					continue
				}
				s.logger.Info("Node status updated to close", "node", node.NodeName, "ip", ip, "port", port)
			}
		}
	}

	return nil
}

// RecordNodeTraffic 记录节点流量信息
func (s *nodeTrafficService) RecordNodeTraffic(ctx context.Context) error {
	// 获取所有节点
	nodes, err := s.nodeRepo.List(ctx, 0, 10000)
	if err != nil {
		s.logger.Error("Failed to get nodes list", "error", err)
		return err
	}

	// 获取当前日期
	currentDate := time.Now().Format("2006-01-02")

	// 为每个节点记录流量信息
	for _, node := range nodes {
		// 跳过离线节点
		if node.Status == 0 {
			continue
		}

		// 构建API请求获取节点信息
		// 这里需要实现具体的API请求逻辑，获取节点的流量数据
		// 为了示例，我们假设有以下数据
		nodeInfo, err := s.getNodeNetworkInfo(ctx, node)
		if err != nil {
			s.logger.Error("Failed to get node network info", "node", node.NodeName, "error", err)
			continue
		}

		// 获取该节点最近一次的流量记录
		lastRecord, err := s.nodeTrafficRepo.GetLastRecord(ctx, node.NodeName)
		if err != nil {
			s.logger.Error("Failed to get last traffic record", "node", node.NodeName, "error", err)
			continue
		}

		// 计算流量增量
		var trafficInIncrement, trafficOutIncrement int64
		if lastRecord != nil {
			trafficInIncrement = max(0, nodeInfo.TotalTrafficIn-lastRecord.TrafficIn)
			trafficOutIncrement = max(0, nodeInfo.TotalTrafficOut-lastRecord.TrafficOut)
		} else {
			// 如果是首次记录，增量就是当前值
			trafficInIncrement = nodeInfo.TotalTrafficIn
			trafficOutIncrement = nodeInfo.TotalTrafficOut
		}

		// 处理当天的增量记录
		todayIncrementRecord, err := s.nodeTrafficRepo.GetTodayIncrement(ctx, node.NodeName, currentDate)
		if err != nil {
			s.logger.Error("Failed to get today's increment record", "node", node.NodeName, "error", err)
			continue
		}

		if todayIncrementRecord != nil {
			// 已有记录，更新
			err = s.nodeTrafficRepo.UpdateIncrement(ctx, todayIncrementRecord.ID, trafficInIncrement, trafficOutIncrement, nodeInfo.OnlineCount)
			if err != nil {
				s.logger.Error("Failed to update increment record", "node", node.NodeName, "error", err)
				continue
			}
		} else {
			// 没有记录，创建新的
			newRecord := &repository.NodeTrafficLog{
				NodeName:    node.NodeName,
				TrafficIn:   trafficInIncrement,
				TrafficOut:  trafficOutIncrement,
				OnlineCount: nodeInfo.OnlineCount,
				RecordTime:  time.Now(),
				RecordDate:  currentDate,
				IsIncrement: true,
			}
			err = s.nodeTrafficRepo.Create(ctx, newRecord)
			if err != nil {
				s.logger.Error("Failed to create increment record", "node", node.NodeName, "error", err)
				continue
			}
		}

		// 处理当天的总流量记录
		todayTotalRecord, err := s.nodeTrafficRepo.GetTodayTotal(ctx, node.NodeName, currentDate)
		if err != nil {
			s.logger.Error("Failed to get today's total record", "node", node.NodeName, "error", err)
			continue
		}

		if todayTotalRecord != nil {
			// 已有记录，更新
			err = s.nodeTrafficRepo.UpdateTotal(ctx, todayTotalRecord.ID, nodeInfo.TotalTrafficIn, nodeInfo.TotalTrafficOut, nodeInfo.OnlineCount)
			if err != nil {
				s.logger.Error("Failed to update total record", "node", node.NodeName, "error", err)
				continue
			}
		} else {
			// 没有记录，创建新的
			newRecord := &repository.NodeTrafficLog{
				NodeName:    node.NodeName,
				TrafficIn:   nodeInfo.TotalTrafficIn,
				TrafficOut:  nodeInfo.TotalTrafficOut,
				OnlineCount: nodeInfo.OnlineCount,
				RecordTime:  time.Now(),
				RecordDate:  currentDate,
				IsIncrement: false,
			}
			err = s.nodeTrafficRepo.Create(ctx, newRecord)
			if err != nil {
				s.logger.Error("Failed to create total record", "node", node.NodeName, "error", err)
				continue
			}
		}

		s.logger.Info("Successfully recorded node traffic", "node", node.NodeName)
	}

	return nil
}

// NodeNetworkInfo 节点网络信息
type NodeNetworkInfo struct {
	TotalTrafficIn  int64
	TotalTrafficOut int64
	OnlineCount     int
}

// getNodeNetworkInfo 获取节点的网络信息
func (s *nodeTrafficService) getNodeNetworkInfo(ctx context.Context, node *repository.Node) (*NodeNetworkInfo, error) {
	// 实际通过API请求获取节点的网络信息

	// 构建节点API URL
	apiURL := node.URL + "/api/serverinfo"

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置Basic认证
	req.SetBasicAuth(node.User, node.Token)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var nodeInfo struct {
		Version         string         `json:"version"`
		BindPort        int            `json:"bindPort"`
		TotalTrafficIn  int64          `json:"totalTrafficIn"`
		TotalTrafficOut int64          `json:"totalTrafficOut"`
		CurConns        int            `json:"curConns"`
		ClientCounts    int            `json:"clientCounts"`
		ProxyTypeCount  map[string]int `json:"proxyTypeCount"`
	}

	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, fmt.Errorf("解析响应数据失败: %w", err)
	}

	// 返回提取的网络信息
	return &NodeNetworkInfo{
		TotalTrafficIn:  nodeInfo.TotalTrafficIn,
		TotalTrafficOut: nodeInfo.TotalTrafficOut,
		OnlineCount:     nodeInfo.ClientCounts,
	}, nil
}
