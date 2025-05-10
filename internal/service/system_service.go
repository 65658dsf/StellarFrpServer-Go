package service

import (
	"context"
	"encoding/json"
	"stellarfrp/internal/model"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

// SystemService 系统状态服务
type SystemService struct {
	systemRepo  *repository.SystemRepository
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewSystemService 创建系统状态服务实例
func NewSystemService(systemRepo *repository.SystemRepository, redisClient *redis.Client, logger *logger.Logger) *SystemService {
	return &SystemService{
		systemRepo:  systemRepo,
		redisClient: redisClient,
		logger:      logger,
	}
}

// GetSystemStatus 获取系统状态
func (s *SystemService) GetSystemStatus(ctx context.Context) (*model.SystemStatus, error) {
	// 尝试从缓存获取
	cacheKey := "system:status"
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var status model.SystemStatus
		if err := json.Unmarshal(cachedData, &status); err == nil {
			return &status, nil
		}
	}

	// 缓存未命中，从数据库获取
	status, err := s.systemRepo.GetSystemStatus(ctx)
	if err != nil {
		s.logger.Error("获取系统状态失败", "error", err)
		return nil, err
	}

	// 将结果存入缓存
	if data, err := json.Marshal(status); err == nil {
		s.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return status, nil
}
