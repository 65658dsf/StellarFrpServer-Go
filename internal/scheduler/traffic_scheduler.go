package scheduler

import (
	"context"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"time"
)

// TrafficScheduler 流量记录调度器
type TrafficScheduler struct {
	userTrafficService service.UserTrafficLogService
	logger             *logger.Logger
	quit               chan struct{}
}

// NewTrafficScheduler 创建流量记录调度器实例
func NewTrafficScheduler(
	userTrafficService service.UserTrafficLogService,
	logger *logger.Logger,
) *TrafficScheduler {
	return &TrafficScheduler{
		userTrafficService: userTrafficService,
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	s.logger.Info("开始记录用户流量")
	err := s.userTrafficService.RecordAllUserTraffic(ctx)
	if err != nil {
		s.logger.Error("用户流量记录失败", "error", err)
	} else {
		s.logger.Info("用户流量记录完成")
	}
}
