package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserCheckinService 用户签到服务接口
type UserCheckinService interface {
	// 用户签到
	Checkin(ctx context.Context, userID int64) (*repository.UserCheckinLog, error)

	// 获取用户签到记录及总数
	GetCheckinLogsWithTotal(ctx context.Context, userID int64, page, pageSize int) ([]*repository.UserCheckinLog, int, error)

	// 检查用户今日是否已签到
	HasCheckedToday(ctx context.Context, userID int64) (bool, error)

	// 获取今日签到统计
	GetTodayStats(ctx context.Context) (int, error)
}

// userCheckinService 用户签到服务实现
type userCheckinService struct {
	userRepo        repository.UserRepository
	groupRepo       repository.GroupRepository
	userCheckinRepo repository.UserCheckinRepository
	redisCli        *redis.Client
	logger          *logger.Logger
}

// NewUserCheckinService 创建用户签到服务实例
func NewUserCheckinService(
	userRepo repository.UserRepository,
	groupRepo repository.GroupRepository,
	userCheckinRepo repository.UserCheckinRepository,
	redisCli *redis.Client,
	logger *logger.Logger,
) UserCheckinService {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	return &userCheckinService{
		userRepo:        userRepo,
		groupRepo:       groupRepo,
		userCheckinRepo: userCheckinRepo,
		redisCli:        redisCli,
		logger:          logger,
	}
}

// Checkin 用户签到
func (s *userCheckinService) Checkin(ctx context.Context, userID int64) (*repository.UserCheckinLog, error) {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户信息失败", "error", err)
		return nil, errors.New("获取用户信息失败")
	}

	// 检查用户是否已经签到
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// 检查用户今天是否已经签到
	checkinLog, err := s.userCheckinRepo.GetByUserAndDate(ctx, userID, todayZero)
	if err != nil {
		s.logger.Error("检查用户签到状态失败", "error", err)
		return nil, errors.New("检查签到状态失败")
	}

	if checkinLog != nil {
		return nil, errors.New("您今天已经签到过了")
	}

	// 获取用户组信息和签到奖励范围
	group, err := s.groupRepo.GetByID(ctx, user.GroupID)
	if err != nil {
		s.logger.Error("获取用户组信息失败", "error", err)
		return nil, errors.New("获取用户组信息失败")
	}

	// 计算连续签到天数
	continuityDays := 1
	if user.LastCheckin != nil {
		yesterday := todayZero.AddDate(0, 0, -1)
		if user.LastCheckin.Year() == yesterday.Year() &&
			user.LastCheckin.Month() == yesterday.Month() &&
			user.LastCheckin.Day() == yesterday.Day() {
			// 如果最后签到时间是昨天，连续签到天数+1
			continuityDays = user.ContinuityCheckin + 1
		} else {
			// 否则重置为1
			continuityDays = 1
		}
	}

	// 计算签到奖励 (在最小和最大之间随机)
	minReward := group.CheckinMinTraffic
	maxReward := group.CheckinMaxTraffic

	// 验证奖励范围
	if minReward <= 0 {
		minReward = 1073741824 // 默认最小1GB
	}
	if maxReward <= 0 || maxReward < minReward {
		maxReward = minReward * 3 // 默认最大为最小的3倍
	}

	// 生成随机奖励
	rewardTraffic := minReward
	if maxReward > minReward {
		rewardTraffic = minReward + rand.Int63n(maxReward-minReward+1)
	}

	// 连续签到奖励 (每多签到1天，额外增加5%的奖励，最多增加50%)
	bonusMultiplier := 1.0
	if continuityDays > 1 {
		// 计算连续签到的额外奖励，每天增加5%，最多增加50%
		extraBonus := float64(continuityDays-1) * 0.05
		if extraBonus > 0.5 {
			extraBonus = 0.5 // 最大50%
		}
		bonusMultiplier = 1.0 + extraBonus
	}
	rewardTraffic = int64(float64(rewardTraffic) * bonusMultiplier)

	// 创建签到记录
	checkinLog = &repository.UserCheckinLog{
		UserID:         userID,
		Username:       user.Username,
		CheckinDate:    todayZero,
		RewardTraffic:  rewardTraffic,
		ContinuityDays: continuityDays,
		CreatedAt:      time.Now(),
	}

	err = s.userCheckinRepo.Create(ctx, checkinLog)
	if err != nil {
		s.logger.Error("创建签到记录失败", "error", err)
		return nil, errors.New("签到失败，请稍后重试")
	}

	// 更新用户信息
	user.LastCheckin = &todayZero
	user.CheckinCount++
	user.ContinuityCheckin = continuityDays

	// 更新用户流量配额
	if user.TrafficQuota == nil {
		initialQuota := rewardTraffic
		user.TrafficQuota = &initialQuota
	} else {
		newQuota := *user.TrafficQuota + rewardTraffic
		user.TrafficQuota = &newQuota
	}

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		s.logger.Error("更新用户信息失败", "error", err)
		return nil, errors.New("更新用户信息失败")
	}

	// 签到成功后清除该用户的签到历史缓存
	s.clearUserCheckinCache(ctx, userID)

	return checkinLog, nil
}

// 生成用户签到历史缓存键
func (s *userCheckinService) getUserCheckinCacheKey(userID int64, page, pageSize int) string {
	return fmt.Sprintf("user:checkin:%d:page:%d:size:%d", userID, page, pageSize)
}

// 清除用户签到历史缓存
func (s *userCheckinService) clearUserCheckinCache(ctx context.Context, userID int64) {
	pattern := fmt.Sprintf("user:checkin:%d:page:*", userID)
	keys, err := s.redisCli.Keys(ctx, pattern).Result()
	if err != nil {
		s.logger.Error("清除签到缓存失败", "error", err, "userID", userID)
		return
	}

	if len(keys) > 0 {
		err = s.redisCli.Del(ctx, keys...).Err()
		if err != nil {
			s.logger.Error("删除签到缓存键失败", "error", err, "userID", userID)
		}
	}
}

// 检查用户今日是否已签到
func (s *userCheckinService) HasCheckedToday(ctx context.Context, userID int64) (bool, error) {
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	checkinLog, err := s.userCheckinRepo.GetByUserAndDate(ctx, userID, todayZero)
	if err != nil {
		return false, err
	}

	return checkinLog != nil, nil
}

// 获取今日签到统计
func (s *userCheckinService) GetTodayStats(ctx context.Context) (int, error) {
	return s.userCheckinRepo.GetTodayCheckinCount(ctx)
}

// 用于缓存的签到记录结构
type checkinLogCache struct {
	Logs  []*repository.UserCheckinLog `json:"logs"`
	Total int                          `json:"total"`
}

// GetCheckinLogsWithTotal 获取用户签到记录及总数
func (s *userCheckinService) GetCheckinLogsWithTotal(ctx context.Context, userID int64, page, pageSize int) ([]*repository.UserCheckinLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 尝试从缓存中获取
	cacheKey := s.getUserCheckinCacheKey(userID, page, pageSize)
	cachedData, err := s.redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，解析数据
		var cacheResult checkinLogCache
		if err := json.Unmarshal([]byte(cachedData), &cacheResult); err == nil {
			return cacheResult.Logs, cacheResult.Total, nil
		}
		// 解析失败，记录日志
		s.logger.Error("解析签到缓存数据失败", "error", err, "userID", userID)
	} else if err.Error() != "redis: nil" {
		// 发生了除缓存不存在之外的错误
		s.logger.Error("获取签到缓存失败", "error", err, "userID", userID)
	}

	// 缓存未命中或出错，从数据库获取
	offset := (page - 1) * pageSize
	logs, total, err := s.userCheckinRepo.GetByUserIDWithTotal(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	// 将结果存入缓存
	cacheData := checkinLogCache{
		Logs:  logs,
		Total: total,
	}
	cacheBytes, err := json.Marshal(cacheData)
	if err == nil {
		// 设置缓存，过期时间30分钟
		err = s.redisCli.Set(ctx, cacheKey, cacheBytes, 30*time.Minute).Err()
		if err != nil {
			s.logger.Error("设置签到缓存失败", "error", err, "userID", userID)
		}
	}

	return logs, total, nil
}
