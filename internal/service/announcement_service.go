package service

import (
	"context"
	"encoding/json"
	"fmt"
	"stellarfrp/internal/model"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

// AnnouncementService 公告服务
type AnnouncementService struct {
	announcementRepo *repository.AnnouncementRepository
	redisClient      *redis.Client
	logger           *logger.Logger
}

// NewAnnouncementService 创建公告服务实例
func NewAnnouncementService(announcementRepo *repository.AnnouncementRepository, redisClient *redis.Client, logger *logger.Logger) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo: announcementRepo,
		redisClient:      redisClient,
		logger:           logger,
	}
}

// GetAnnouncements 获取分页公告列表
func (s *AnnouncementService) GetAnnouncements(ctx context.Context, page, limit int) (*model.PaginatedAnnouncements, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("announcements:list:%d:%d", page, limit)
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var result model.PaginatedAnnouncements
		if err := json.Unmarshal(cachedData, &result); err == nil {
			return &result, nil
		}
	}

	// 缓存未命中，从数据库获取
	total, err := s.announcementRepo.CountAnnouncements(ctx)
	if err != nil {
		s.logger.Error("获取公告总数失败", "error", err)
		return nil, err
	}

	announcements, err := s.announcementRepo.GetAnnouncements(ctx, page, limit)
	if err != nil {
		s.logger.Error("获取公告列表失败", "error", err)
		return nil, err
	}

	result := &model.PaginatedAnnouncements{
		Total: total,
		Items: announcements,
	}

	// 将结果存入缓存
	if data, err := json.Marshal(result); err == nil {
		s.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return result, nil
}

// GetAnnouncementByID 根据ID获取公告详情
func (s *AnnouncementService) GetAnnouncementByID(ctx context.Context, id int64) (*model.Announcement, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("announcements:detail:%d", id)
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var announcement model.Announcement
		if err := json.Unmarshal(cachedData, &announcement); err == nil {
			return &announcement, nil
		}
	}

	// 缓存未命中，从数据库获取
	announcement, err := s.announcementRepo.GetAnnouncementByID(ctx, id)
	if err != nil {
		s.logger.Error("获取公告详情失败", "id", id, "error", err)
		return nil, err
	}

	// 将结果存入缓存
	if data, err := json.Marshal(announcement); err == nil {
		s.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return announcement, nil
}

// InvalidateCache 使缓存失效
func (s *AnnouncementService) InvalidateCache(ctx context.Context) error {
	// 删除所有公告相关的缓存
	pattern := "announcements:*"
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			s.logger.Error("删除缓存失败", "key", iter.Val(), "error", err)
		}
	}
	return iter.Err()
}

// CreateAnnouncement 创建公告
func (s *AnnouncementService) CreateAnnouncement(ctx context.Context, a *model.Announcement) error {
	err := s.announcementRepo.CreateAnnouncement(ctx, a)
	if err == nil {
		s.InvalidateCache(ctx)
	}
	return err
}

// UpdateAnnouncement 更新公告
func (s *AnnouncementService) UpdateAnnouncement(ctx context.Context, a *model.Announcement) error {
	a.PublishDate = time.Now() // 更新发布时间为当前时间
	err := s.announcementRepo.UpdateAnnouncement(ctx, a)
	if err == nil {
		s.InvalidateCache(ctx)
	}
	return err
}

// DeleteAnnouncement 删除公告
func (s *AnnouncementService) DeleteAnnouncement(ctx context.Context, id int64) error {
	err := s.announcementRepo.DeleteAnnouncement(ctx, id)
	if err == nil {
		s.InvalidateCache(ctx)
	}
	return err
}

// GetAnnouncementsAdmin 管理员获取所有公告（含不可见）
func (s *AnnouncementService) GetAnnouncementsAdmin(ctx context.Context, page, limit int) (*model.PaginatedAnnouncements, error) {
	total, err := s.announcementRepo.CountAnnouncementsAdmin(ctx)
	if err != nil {
		s.logger.Error("获取公告总数失败", "error", err)
		return nil, err
	}
	announcements, err := s.announcementRepo.GetAnnouncementsAdmin(ctx, page, limit)
	if err != nil {
		s.logger.Error("获取公告列表失败", "error", err)
		return nil, err
	}
	return &model.PaginatedAnnouncements{
		Total: total,
		Items: announcements,
	}, nil
}

// GetAnnouncementByIDAdmin 管理员获取单个公告（含不可见）
func (s *AnnouncementService) GetAnnouncementByIDAdmin(ctx context.Context, id int64) (*model.Announcement, error) {
	return s.announcementRepo.GetAnnouncementByID(ctx, id)
}
