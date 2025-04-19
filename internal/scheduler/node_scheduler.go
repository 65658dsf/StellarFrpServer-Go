package scheduler

import (
	"context"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"time"
)

// NodeScheduler 节点调度器
type NodeScheduler struct {
	nodeTrafficService service.NodeTrafficService
	userTrafficService service.UserTrafficService
	logger             *logger.Logger
	quit               chan struct{}
}

// NewNodeScheduler 创建节点调度器实例
func NewNodeScheduler(
	nodeTrafficService service.NodeTrafficService,
	userTrafficService service.UserTrafficService,
	logger *logger.Logger,
) *NodeScheduler {
	return &NodeScheduler{
		nodeTrafficService: nodeTrafficService,
		userTrafficService: userTrafficService,
		logger:             logger,
		quit:               make(chan struct{}),
	}
}

// Start 启动节点调度器
func (s *NodeScheduler) Start() {
	// 启动定时检查节点状态的goroutine
	go s.checkNodeStatusScheduler()

	// 启动定时记录节点流量的goroutine
	go s.recordNodeTrafficScheduler()

	// 启动定时记录用户流量的goroutine
	go s.recordUserTrafficScheduler()

	s.logger.Info("节点调度器启动")
}

// Stop 停止节点调度器
func (s *NodeScheduler) Stop() {
	close(s.quit)
	s.logger.Info("节点调度器停止")
}

// checkNodeStatusScheduler 节点状态检查定时器
func (s *NodeScheduler) checkNodeStatusScheduler() {
	// 立即运行一次检查
	s.checkNodeStatus()

	// 创建一个定时器，每10分钟检查一次节点状态
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkNodeStatus()
		case <-s.quit:
			return
		}
	}
}

// recordNodeTrafficScheduler 节点流量记录定时器
func (s *NodeScheduler) recordNodeTrafficScheduler() {
	// 计算到当天23:55的时间
	now := time.Now()
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 55, 0, 0, now.Location())
	if now.After(nextRunTime) {
		// 如果当前时间已经过了今天的23:55，则设置为明天的23:55
		nextRunTime = nextRunTime.Add(24 * time.Hour)
	}

	// 计算到下次运行的时间间隔
	initialDelay := nextRunTime.Sub(now)

	// 等待到首次运行时间
	time.Sleep(initialDelay)

	// 运行一次流量记录
	s.recordNodeTraffic()

	// 创建一个定时器，每天运行一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.recordNodeTraffic()
		case <-s.quit:
			return
		}
	}
}

// recordUserTrafficScheduler 用户隧道流量记录定时器
func (s *NodeScheduler) recordUserTrafficScheduler() {
	// 计算到当天23:55的时间
	now := time.Now()
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 55, 0, 0, now.Location())
	if now.After(nextRunTime) {
		// 如果当前时间已经过了今天的23:55，则设置为明天的23:55
		nextRunTime = nextRunTime.Add(24 * time.Hour)
	}

	// 计算到下次运行的时间间隔
	initialDelay := nextRunTime.Sub(now)

	// 等待到首次运行时间
	time.Sleep(initialDelay)

	// 运行一次用户流量记录
	s.recordUserTraffic()

	// 创建一个定时器，每天运行一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.recordUserTraffic()
		case <-s.quit:
			return
		}
	}
}

// checkNodeStatus 检查节点状态的具体实现
func (s *NodeScheduler) checkNodeStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	err := s.nodeTrafficService.CheckNodeStatus(ctx)
	if err != nil {
		s.logger.Error("节点状态检查失败", "error", err)
	} else {
		s.logger.Info("节点状态检查完成")
	}
}

// recordNodeTraffic 记录节点流量的具体实现
func (s *NodeScheduler) recordNodeTraffic() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := s.nodeTrafficService.RecordNodeTraffic(ctx)
	if err != nil {
		s.logger.Error("节点流量记录失败", "error", err)
	} else {
		s.logger.Info("节点流量记录完成")
	}
}

// recordUserTraffic 记录用户隧道流量的具体实现
func (s *NodeScheduler) recordUserTraffic() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := s.userTrafficService.RecordUserTraffic(ctx)
	if err != nil {
		s.logger.Error("用户隧道流量记录失败", "error", err)
	} else {
		s.logger.Info("用户隧道流量记录完成")
	}
}
