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

// AdService 广告服务
type AdService struct {
	adRepo      *repository.AdRepository
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewAdService 创建广告服务实例
func NewAdService(adRepo *repository.AdRepository, redisClient *redis.Client, logger *logger.Logger) *AdService {
	return &AdService{
		adRepo:      adRepo,
		redisClient: redisClient,
		logger:      logger,
	}
}

// GetAds 获取广告列表
func (s *AdService) GetAds(ctx context.Context) ([]model.Ad, error) {
	// 尝试从缓存获取
	cacheKey := "ads:list"
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var ads []model.Ad
		if err := json.Unmarshal(cachedData, &ads); err == nil {
			return ads, nil
		}
	}

	// 缓存未命中，从数据库获取
	ads, err := s.adRepo.GetAds(ctx)
	if err != nil {
		s.logger.Error("获取广告列表失败", "error", err)
		return nil, err
	}

	// 将结果存入缓存
	if data, err := json.Marshal(ads); err == nil {
		s.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return ads, nil
}

// InvalidateCache 使缓存失效
func (s *AdService) InvalidateCache(ctx context.Context) error {
	// 删除所有广告相关的缓存
	pattern := "ads:*"
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			s.logger.Error("删除缓存失败", "key", iter.Val(), "error", err)
		}
	}
	return iter.Err()
}

// GetActiveAds 获取所有活跃的广告
func (s *AdService) GetActiveAds(ctx context.Context) ([]*model.Ad, error) {
	ads, err := s.adRepo.GetActiveAds(ctx)
	if err != nil {
		s.logger.Error("获取广告失败", "error", err)
		return nil, err
	}
	return ads, nil
}
