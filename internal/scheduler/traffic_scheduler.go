package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"time"
)

// TrafficScheduler 流量记录调度器
type TrafficScheduler struct {
	userTrafficService service.UserTrafficLogService
	userService        service.UserService
	proxyService       service.ProxyService
	nodeService        service.NodeService
	logger             *logger.Logger
	quit               chan struct{}
}

// NewTrafficScheduler 创建流量记录调度器实例
func NewTrafficScheduler(
	userTrafficService service.UserTrafficLogService,
	userService service.UserService,
	proxyService service.ProxyService,
	nodeService service.NodeService,
	logger *logger.Logger,
) *TrafficScheduler {
	return &TrafficScheduler{
		userTrafficService: userTrafficService,
		userService:        userService,
		proxyService:       proxyService,
		nodeService:        nodeService,
		logger:             logger,
		quit:               make(chan struct{}),
	}
}

// Start 启动流量记录调度器
func (s *TrafficScheduler) Start() {
	// 启动时立即记录一次用户流量
	go s.recordUserTraffic()

	// 启动定时记录用户流量的goroutine
	go s.scheduleUserTrafficRecording()

	s.logger.Info("流量记录调度器启动")
}

// Stop 停止流量记录调度器
func (s *TrafficScheduler) Stop() {
	close(s.quit)
	s.logger.Info("流量记录调度器停止")
}

// scheduleUserTrafficRecording 用户流量记录定时器
func (s *TrafficScheduler) scheduleUserTrafficRecording() {
	// 计算到当天23:50的时间
	now := time.Now()
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 50, 0, 0, now.Location())
	if now.After(nextRunTime) {
		// 如果当前时间已经过了今天的23:50，则设置为明天的23:50
		nextRunTime = nextRunTime.Add(24 * time.Hour)
	}

	// 计算到下次运行的时间间隔
	initialDelay := nextRunTime.Sub(now)
	s.logger.Info("用户流量记录计划", "nextRunTime", nextRunTime.Format("2006-01-02 15:04:05"))

	// 等待到首次运行时间
	select {
	case <-time.After(initialDelay):
		s.recordUserTraffic()
	case <-s.quit:
		return
	}

	// 创建一个定时器，每天23:50运行一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			// 计算今天23:50的时间
			targetTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 50, 0, 0, now.Location())
			// 如果现在已经过了23:50，就立即执行，否则等到23:50
			if now.After(targetTime) {
				s.recordUserTraffic()
			} else {
				delay := targetTime.Sub(now)
				time.Sleep(delay)
				s.recordUserTraffic()
			}
		case <-s.quit:
			return
		}
	}
}

// recordUserTraffic 记录用户流量
func (s *TrafficScheduler) recordUserTraffic() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.logger.Info("开始记录用户流量任务")
	err := s.userTrafficService.RecordAllUserTraffic(ctx)
	if err != nil {
		s.logger.Error("用户流量记录失败", "error", err)
	} else {
		s.logger.Info("用户流量记录完成")
	}

	// -- 检查白银用户组 (ID=3) 是否过期 --
	silverGroupID := int64(3)
	defaultGroupIDAfterExpiry := int64(2) // 普通用户组ID

	expiredSilverUsers, err := s.userService.GetUsersWithExpiredGroup(ctx, silverGroupID)
	if err != nil {
		s.logger.Error("获取白银用户组已过期用户列表失败", "groupID", silverGroupID, "error", err)
	} else {
		if len(expiredSilverUsers) > 0 {
			s.logger.Info("检测到白银用户组有权限过期的用户", "groupID", silverGroupID, "count", len(expiredSilverUsers))
			for _, user := range expiredSilverUsers {
				originalGroupID := user.GroupID // 应该是3
				user.GroupID = defaultGroupIDAfterExpiry
				user.GroupTime = nil
				if errUpdate := s.userService.Update(ctx, user); errUpdate != nil {
					s.logger.Error("更新白银用户组过期用户失败", "username", user.Username, "original_group_id", originalGroupID, "error", errUpdate)
				} else {
					s.logger.Info("白银用户组已到期，成功重置为普通用户组", "username", user.Username, "original_group_id", originalGroupID, "new_group_id", user.GroupID)
				}
			}
		}
	}
	// -- 白银用户组过期检查结束 --

	// 获取所有用户 (用于后续的流量超额检查，这一部分逻辑暂时保留，如果希望也针对特定用户组，可以进一步优化)
	allUsersForTrafficCheck, err := s.userService.GetAllUsers(ctx)
	if err != nil {
		s.logger.Error("获取所有用户列表以进行流量检查失败", "error", err)
		// 如果此处失败，后续的流量检查将无法进行
	} else {
		for _, user := range allUsersForTrafficCheck {
			// 2. 检查用户流量是否超额
			userTrafficLog, err := s.userTrafficService.GetUserTodayTraffic(ctx, user.Username)
			if err != nil {
				s.logger.Error("获取用户流量记录失败", "username", user.Username, "error", err)
				continue
			}

			totalTrafficQuotaBytes := int64(0)
			userGroup, err := s.userService.GetUserGroup(ctx, user.ID)
			if err != nil {
				s.logger.Error("获取用户组信息失败", "username", user.Username, "error", err)
			} else if userGroup != nil {
				totalTrafficQuotaBytes += userGroup.TrafficQuota
			}
			if user.TrafficQuota != nil {
				totalTrafficQuotaBytes += *user.TrafficQuota
			}

			if totalTrafficQuotaBytes > 0 && userTrafficLog.TotalTraffic >= totalTrafficQuotaBytes {
				s.logger.Warn("用户总流量已超额，准备关闭其所有隧道", "username", user.Username, "used_total_bytes", userTrafficLog.TotalTraffic, "quota_bytes", totalTrafficQuotaBytes)

				userProxies, err := s.proxyService.GetByUsername(ctx, user.Username)
				if err != nil {
					s.logger.Error("获取用户隧道列表失败，无法关闭超额隧道", "username", user.Username, "error", err)
					continue
				}

				closedCount := 0
				for _, proxy := range userProxies {
					if proxy.Status == "online" && proxy.RunID != "" {
						s.logger.Info("准备关闭超额用户的隧道", "username", user.Username, "proxy_name", proxy.ProxyName, "run_id", proxy.RunID)
						if err := s.closeTunnel(ctx, proxy); err != nil {
							s.logger.Error("关闭隧道失败", "username", user.Username, "proxy_name", proxy.ProxyName, "run_id", proxy.RunID, "error", err)
						} else {
							closedCount++
							proxy.Status = "offline"
							proxy.RunID = ""
							proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
							if errUpdate := s.proxyService.Update(ctx, proxy); errUpdate != nil {
								s.logger.Error("更新隧道状态为offline失败", "username", user.Username, "proxy_name", proxy.ProxyName, "error", errUpdate)
							}
						}
					}
				}
				if closedCount > 0 {
					s.logger.Info("用户因流量超额，隧道已关闭", "username", user.Username, "closed_tunnel_count", closedCount, "total_proxies", len(userProxies))
				}
			}
		}
	}
	s.logger.Info("所有用户检查完成")
}

// closeTunnel 关闭单个隧道，通过调用节点API
func (s *TrafficScheduler) closeTunnel(ctx context.Context, proxy *repository.Proxy) error {
	if proxy.RunID == "" {
		s.logger.Info("隧道没有有效的 RunID，无需关闭", "proxy_name", proxy.ProxyName)
		return nil // 没有运行ID，无需关闭
	}

	node, err := s.nodeService.GetByID(ctx, proxy.Node)
	if err != nil {
		return fmt.Errorf("获取节点信息失败: %w", err)
	}
	if node == nil {
		return fmt.Errorf("节点不存在: %d", proxy.Node)
	}

	apiURL := fmt.Sprintf("%s/api/client/kick", node.URL)
	requestBody, err := json.Marshal(map[string]string{"runid": proxy.RunID})
	if err != nil {
		return fmt.Errorf("构建请求体失败: %w", err)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(node.User, node.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送关闭隧道请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("节点返回错误, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	s.logger.Info("成功向节点发送关闭隧道请求", "proxy_name", proxy.ProxyName, "run_id", proxy.RunID, "node_url", apiURL)
	return nil
}
