package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"sync"
	"time"
)

// UserTrafficService 用户流量服务接口
type UserTrafficService interface {
	RecordUserTraffic(ctx context.Context) error
}

// userTrafficService 用户流量服务实现
type userTrafficService struct {
	userRepo        repository.UserRepository
	proxyRepo       repository.ProxyRepository
	nodeRepo        repository.NodeRepository
	groupRepo       repository.GroupRepository
	userTrafficRepo repository.UserTrafficRepository
	logger          *logger.Logger
}

// NewUserTrafficService 创建用户流量服务实例
func NewUserTrafficService(
	userRepo repository.UserRepository,
	proxyRepo repository.ProxyRepository,
	nodeRepo repository.NodeRepository,
	groupRepo repository.GroupRepository,
	userTrafficRepo repository.UserTrafficRepository,
	logger *logger.Logger,
) UserTrafficService {
	return &userTrafficService{
		userRepo:        userRepo,
		proxyRepo:       proxyRepo,
		nodeRepo:        nodeRepo,
		groupRepo:       groupRepo,
		userTrafficRepo: userTrafficRepo,
		logger:          logger,
	}
}

// RecordUserTraffic 记录用户隧道流量信息
func (s *userTrafficService) RecordUserTraffic(ctx context.Context) error {
	// 获取当前日期
	currentDate := time.Now().Format("2006-01-02")

	// 获取所有用户
	users, err := s.userRepo.List(ctx, 0, 10000)
	if err != nil {
		s.logger.Error("获取用户列表失败", "error", err)
		return err
	}

	// 为每个用户记录流量信息
	for _, user := range users {
		// 获取用户个人流量配额
		var userQuota int64 = 0
		if user.TrafficQuota != nil {
			userQuota = *user.TrafficQuota
		}

		// 获取用户组流量配额
		var groupQuota int64 = 0
		if user.GroupID > 0 {
			group, err := s.groupRepo.GetByID(ctx, user.GroupID)
			if err != nil {
				s.logger.Error("获取用户组信息失败", "groupID", user.GroupID, "error", err)
			} else if group != nil {
				groupQuota = group.TrafficQuota
			}
		}

		// 计算用户总流量配额 (用户配额 + 用户组配额)
		trafficQuota := userQuota + groupQuota

		// 获取用户所有的隧道
		proxies, err := s.proxyRepo.GetByUsername(ctx, user.Username)
		if err != nil {
			s.logger.Error("获取用户隧道列表失败", "username", user.Username, "error", err)
			continue
		}

		// 如果用户没有隧道，跳过
		if len(proxies) == 0 {
			continue
		}

		// 根据节点对隧道进行分组
		nodeProxiesMap := make(map[int64][]*repository.Proxy) // 节点ID -> 隧道列表
		for _, proxy := range proxies {
			nodeProxiesMap[proxy.Node] = append(nodeProxiesMap[proxy.Node], proxy)
		}

		// 用于存储所有隧道的流量数据
		proxyTrafficMap := make(map[int64]int64) // 隧道ID -> 流量总和(in + out)

		// 创建HTTP客户端，设置超时
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// 用于并发请求的等待组
		var wg sync.WaitGroup
		var mu sync.Mutex

		// 并发请求每个节点的API获取隧道状态
		for nodeID, nodeProxies := range nodeProxiesMap {
			wg.Add(1)
			go func(nodeID int64, nodeProxies []*repository.Proxy) {
				defer wg.Done()

				// 获取节点信息
				node, err := s.nodeRepo.GetByID(ctx, nodeID)
				if err != nil {
					s.logger.Error("获取节点信息失败", "nodeID", nodeID, "error", err)
					return
				}

				// 构建节点API URL - 请求所有隧道状态
				apiURL := fmt.Sprintf("%s/api/proxy/", node.URL)

				// 创建请求
				req, err := http.NewRequest("GET", apiURL, nil)
				if err != nil {
					s.logger.Error("创建请求失败", "error", err, "url", apiURL)
					return
				}

				// 设置Basic认证
				req.SetBasicAuth(node.User, node.Token)

				// 发送请求
				resp, err := client.Do(req)
				if err != nil {
					s.logger.Error("发送请求失败", "error", err, "nodeID", nodeID)
					return
				}
				defer resp.Body.Close()

				// 读取响应
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					s.logger.Error("读取响应失败", "error", err, "nodeID", nodeID)
					return
				}

				// 解析响应
				var allProxiesStatus map[string]interface{}
				if err := json.Unmarshal(body, &allProxiesStatus); err != nil {
					s.logger.Error("解析响应失败", "error", err, "body", string(body), "nodeID", nodeID)
					return
				}

				// 收集所有类型的隧道
				allProxies := make([]interface{}, 0)
				proxyTypes := []string{"tcp", "udp", "http", "https", "stcp", "xtcp"}

				for _, pType := range proxyTypes {
					if typeProxiesList, ok := allProxiesStatus[pType].(map[string]interface{}); ok {
						if proxiesList, ok := typeProxiesList["proxies"].([]interface{}); ok {
							allProxies = append(allProxies, proxiesList...)
						}
					}
				}

				// 将隧道列表转换为map以便快速查找
				proxyStatusMap := make(map[string]interface{})
				for _, proxyData := range allProxies {
					proxyInfo, ok := proxyData.(map[string]interface{})
					if !ok {
						continue
					}

					name, ok := proxyInfo["name"].(string)
					if !ok {
						continue
					}

					proxyStatusMap[name] = proxyInfo
				}

				// 处理用户在该节点的所有隧道
				for _, proxy := range nodeProxies {
					expectedProxyName := user.Username + "." + proxy.ProxyName

					// 在节点返回的所有代理中查找该代理
					proxyStatus, found := proxyStatusMap[expectedProxyName]
					if found {
						// 解析并提取流量数据
						proxyInfo, ok := proxyStatus.(map[string]interface{})
						if !ok {
							continue
						}

						// 提取流量数据
						var trafficIn, trafficOut int64 = 0, 0
						if in, ok := proxyInfo["todayTrafficIn"].(float64); ok {
							trafficIn = int64(in)
						}
						if out, ok := proxyInfo["todayTrafficOut"].(float64); ok {
							trafficOut = int64(out)
						}

						// 更新隧道流量数据
						mu.Lock()
						proxyTrafficMap[proxy.ID] = trafficIn + trafficOut
						mu.Unlock()
					}
				}
			}(nodeID, nodeProxies)
		}

		// 等待所有请求完成
		wg.Wait()

		// 计算用户总流量
		var todayTotalTraffic int64 = 0
		for _, traffic := range proxyTrafficMap {
			todayTotalTraffic += traffic
		}

		// 如果所有隧道没有流量数据，使用默认方法
		if todayTotalTraffic == 0 {
			for _, proxy := range proxies {
				// 使用配额作为已消耗流量的近似值
				proxyTraffic := proxy.TrafficQuota
				todayTotalTraffic += proxyTraffic
			}
			s.logger.Info("使用隧道配额作为流量近似值", "username", user.Username)
		}

		// 获取用户当前的流量记录
		userTraffic, err := s.userTrafficRepo.GetByUsername(ctx, user.Username, currentDate)
		if err != nil {
			s.logger.Error("获取用户流量记录失败", "username", user.Username, "error", err)
			continue
		}

		// 计算使用百分比
		var usagePercent float64 = 0
		if trafficQuota > 0 {
			// 计算已使用流量百分比(总流量/配额)
			usagePercent = float64(todayTotalTraffic) / float64(trafficQuota) * 100
			if usagePercent > 100 {
				usagePercent = 100 // 最大100%
			}
		}

		// 如果没有当天记录，创建新记录
		if userTraffic == nil {
			userTraffic = &repository.UserTrafficLog{
				Username:     user.Username,
				TotalTraffic: todayTotalTraffic, // 初始总流量
				TodayTraffic: todayTotalTraffic, // 今日流量
				TrafficQuota: trafficQuota,      // 用户总流量配额(个人+用户组)
				UsagePercent: usagePercent,      // 使用百分比
				RecordDate:   currentDate,
			}
		} else {
			// 更新现有记录
			userTraffic.TodayTraffic = todayTotalTraffic
			userTraffic.TotalTraffic += todayTotalTraffic
			userTraffic.TrafficQuota = trafficQuota
			userTraffic.UsagePercent = usagePercent
		}

		// 更新用户流量记录
		err = s.userTrafficRepo.CreateOrUpdate(ctx, userTraffic)
		if err != nil {
			s.logger.Error("更新用户流量记录失败", "username", user.Username, "error", err)
			continue
		}

		// 更新历史流量记录
		err = s.userTrafficRepo.UpdateHistoryTraffic(ctx, user.Username, currentDate, todayTotalTraffic)
		if err != nil {
			s.logger.Error("更新用户历史流量记录失败", "username", user.Username, "error", err)
			continue
		}

		s.logger.Info("成功记录用户流量", "username", user.Username, "todayTraffic", todayTotalTraffic, "usagePercent", usagePercent)
	}

	return nil
}
