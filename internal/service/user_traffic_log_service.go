package service

import (
	"context"
	"encoding/json"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/types"
	"stellarfrp/pkg/logger"
	"strings"
	"sync"
)

// UserTrafficLogService 用户流量日志服务接口
type UserTrafficLogService interface {
	// 记录所有用户的流量信息
	RecordAllUserTraffic(ctx context.Context) error
}

// userTrafficLogService 用户流量日志服务实现
type userTrafficLogService struct {
	nodeRepo        repository.NodeRepository
	userTrafficRepo repository.UserTrafficLogRepository
	httpClient      *http.Client
	logger          *logger.Logger
}

// NewUserTrafficLogService 创建用户流量日志服务实例
func NewUserTrafficLogService(
	nodeRepo repository.NodeRepository,
	userTrafficRepo repository.UserTrafficLogRepository,
	logger *logger.Logger,
) UserTrafficLogService {
	return &userTrafficLogService{
		nodeRepo:        nodeRepo,
		userTrafficRepo: userTrafficRepo,
		httpClient:      &http.Client{Timeout: apiTimeout},
		logger:          logger,
	}
}

// RecordAllUserTraffic 记录所有用户的流量信息
func (s *userTrafficLogService) RecordAllUserTraffic(ctx context.Context) error {
	// 首先确保表存在
	if err := s.userTrafficRepo.EnsureTableExists(ctx); err != nil {
		s.logger.Error("确保表存在失败", "error", err)
		return err
	}

	// 获取所有节点信息
	nodes, err := s.nodeRepo.List(ctx, 0, 1000)
	if err != nil {
		s.logger.Error("获取节点信息失败", "error", err)
		return err
	}

	// 用于累积每个用户的流量
	userTraffic := make(map[string]int64)
	var mutex sync.Mutex

	// 使用WaitGroup等待所有异步任务完成
	var wg sync.WaitGroup

	// 为每种代理类型发送请求
	proxyTypes := []string{"tcp", "udp", "http", "https"}

	// 并发请求节点API
	for _, node := range nodes {
		wg.Add(1)
		go func(node *repository.Node) {
			defer wg.Done()

			// 为当前节点的各种代理类型获取流量数据
			nodeUserTraffic := s.collectNodeTraffic(ctx, node, proxyTypes)

			// 将当前节点的用户流量数据合并到总流量中
			mutex.Lock()
			for username, traffic := range nodeUserTraffic {
				userTraffic[username] += traffic
			}
			mutex.Unlock()
		}(node)
	}

	// 等待所有节点流量收集完成
	wg.Wait()

	// 更新数据库中的用户流量记录
	var updateWg sync.WaitGroup
	for username, todayTraffic := range userTraffic {
		updateWg.Add(1)
		go func(username string, todayTraffic int64) {
			defer updateWg.Done()
			if err := s.userTrafficRepo.UpdateTraffic(ctx, username, todayTraffic); err != nil {
				s.logger.Error("更新用户流量记录失败", "username", username, "error", err)
			}
		}(username, todayTraffic)
	}

	// 等待所有更新完成
	updateWg.Wait()

	s.logger.Info("所有用户流量记录完成", "userCount", len(userTraffic))
	return nil
}

// collectNodeTraffic 收集指定节点的所有代理类型流量数据
func (s *userTrafficLogService) collectNodeTraffic(ctx context.Context, node *repository.Node, proxyTypes []string) map[string]int64 {
	nodeUserTraffic := make(map[string]int64)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, proxyType := range proxyTypes {
		wg.Add(1)
		go func(proxyType string) {
			defer wg.Done()

			// 发送API请求获取流量数据
			data, err := s.getProxyTraffic(ctx, node.URL, node.User, node.Token, proxyType)
			if err != nil {
				s.logger.Error("获取节点流量数据失败",
					"node", node.NodeName,
					"type", proxyType,
					"error", err)
				return
			}

			// 处理返回的数据
			for _, proxy := range data.Proxies {
				if proxy.Name != "" {
					username := extractUsername(proxy.Name)
					// 计算进出流量总和
					totalTraffic := proxy.TodayTrafficIn + proxy.TodayTrafficOut

					mutex.Lock()
					nodeUserTraffic[username] += totalTraffic
					mutex.Unlock()
				}
			}
		}(proxyType)
	}

	wg.Wait()
	return nodeUserTraffic
}

// getProxyTraffic 获取指定类型的代理流量数据
func (s *userTrafficLogService) getProxyTraffic(ctx context.Context, url, user, token, proxyType string) (*types.ProxyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		url+"/api/proxy/"+proxyType, nil)
	if err != nil {
		return nil, err
	}

	// 添加基本认证
	req.SetBasicAuth(user, token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response types.ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// extractUsername 从代理名称中提取用户名
func extractUsername(name string) string {
	if idx := strings.Index(name, "."); idx > 0 {
		return name[:idx]
	}
	return name
}
