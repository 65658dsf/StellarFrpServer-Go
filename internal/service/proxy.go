package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/utils"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

// ProxyService 隧道服务接口
type ProxyService interface {
	Create(ctx context.Context, proxy *repository.Proxy) (int64, error)
	GetByID(ctx context.Context, id int64) (*repository.Proxy, error)
	GetByUsername(ctx context.Context, username string) ([]*repository.Proxy, error)
	GetByUsernameWithPagination(ctx context.Context, username string, offset, limit int) ([]*repository.Proxy, error)
	GetByUsernameAndName(ctx context.Context, username, proxyName string) (*repository.Proxy, error)
	Update(ctx context.Context, proxy *repository.Proxy) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*repository.Proxy, error)
	Count(ctx context.Context) (int, error)
	IsRemotePortUsed(ctx context.Context, nodeID int64, proxyType string, remotePort string) (bool, error)
	GetUserProxyCount(ctx context.Context, username string) (int, error)
	CheckUserNodeAccess(ctx context.Context, username string, nodeID int64) (bool, error)
}

// proxyService 隧道服务实现
type proxyService struct {
	proxyRepo   repository.ProxyRepository
	nodeService NodeService
	userService UserService
	redisCli    *redis.Client
	logger      *logger.Logger
}

// NewProxyService 创建隧道服务实例
func NewProxyService(
	proxyRepo repository.ProxyRepository,
	nodeService NodeService,
	userService UserService,
	redisCli *redis.Client,
	logger *logger.Logger,
) ProxyService {
	return &proxyService{
		proxyRepo:   proxyRepo,
		nodeService: nodeService,
		userService: userService,
		redisCli:    redisCli,
		logger:      logger,
	}
}

// 生成用户隧道列表缓存键
func (s *proxyService) getUserProxiesCacheKey(username string) string {
	return fmt.Sprintf("user:proxies:%s", username)
}

// 生成用户分页隧道列表缓存键
func (s *proxyService) getUserProxiesPaginationCacheKey(username string, offset, limit int) string {
	return fmt.Sprintf("user:proxies:%s:offset:%d:limit:%d", username, offset, limit)
}

// 生成用户隧道数量缓存键
func (s *proxyService) getUserProxyCountCacheKey(username string) string {
	return fmt.Sprintf("user:proxy_count:%s", username)
}

// 清除用户相关的所有隧道缓存
func (s *proxyService) clearUserProxiesCache(ctx context.Context, username string) {
	// 清除用户隧道列表缓存
	listKey := s.getUserProxiesCacheKey(username)
	s.redisCli.Del(ctx, listKey)

	// 清除用户隧道数量缓存
	countKey := s.getUserProxyCountCacheKey(username)
	s.redisCli.Del(ctx, countKey)

	// 清除分页缓存
	pattern := fmt.Sprintf("user:proxies:%s:offset:*", username)
	keys, err := s.redisCli.Keys(ctx, pattern).Result()
	if err != nil {
		s.logger.Error("清除隧道分页缓存失败", "error", err, "username", username)
		return
	}

	if len(keys) > 0 {
		err = s.redisCli.Del(ctx, keys...).Err()
		if err != nil {
			s.logger.Error("删除隧道分页缓存键失败", "error", err, "username", username)
		}
	}
}

// Create 创建隧道
func (s *proxyService) Create(ctx context.Context, proxy *repository.Proxy) (int64, error) {
	id, err := s.proxyRepo.Create(ctx, proxy)
	if err != nil {
		return 0, err
	}

	// 创建成功后清除用户隧道缓存
	s.clearUserProxiesCache(ctx, proxy.Username)
	return id, nil
}

// GetByID 根据ID获取隧道
func (s *proxyService) GetByID(ctx context.Context, id int64) (*repository.Proxy, error) {
	return s.proxyRepo.GetByID(ctx, id)
}

// GetByUsername 根据用户名获取隧道列表
func (s *proxyService) GetByUsername(ctx context.Context, username string) ([]*repository.Proxy, error) {
	// 尝试从缓存获取
	cacheKey := s.getUserProxiesCacheKey(username)
	cachedData, err := s.redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，解析数据
		var proxies []*repository.Proxy
		if err := json.Unmarshal([]byte(cachedData), &proxies); err == nil {
			return proxies, nil
		}
		// 解析失败，记录日志
		s.logger.Error("解析隧道列表缓存数据失败", "error", err, "username", username)
	} else if err.Error() != "redis: nil" {
		// 发生了除缓存不存在之外的错误
		s.logger.Error("获取隧道列表缓存失败", "error", err, "username", username)
	}

	// 缓存未命中或出错，从数据库获取
	proxies, err := s.proxyRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// 将结果存入缓存
	if len(proxies) > 0 {
		cacheBytes, err := json.Marshal(proxies)
		if err == nil {
			// 设置缓存，过期时间30分钟
			err = s.redisCli.Set(ctx, cacheKey, cacheBytes, 30*time.Minute).Err()
			if err != nil {
				s.logger.Error("设置隧道列表缓存失败", "error", err, "username", username)
			}
		}
	}

	return proxies, nil
}

// GetByUsernameWithPagination 根据用户名获取隧道列表（带分页）
func (s *proxyService) GetByUsernameWithPagination(ctx context.Context, username string, offset, limit int) ([]*repository.Proxy, error) {
	// 尝试从缓存获取
	cacheKey := s.getUserProxiesPaginationCacheKey(username, offset, limit)
	cachedData, err := s.redisCli.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，解析数据
		var proxies []*repository.Proxy
		if err := json.Unmarshal([]byte(cachedData), &proxies); err == nil {
			return proxies, nil
		}
		// 解析失败，记录日志
		s.logger.Error("解析分页隧道列表缓存数据失败", "error", err, "username", username)
	} else if err.Error() != "redis: nil" {
		// 发生了除缓存不存在之外的错误
		s.logger.Error("获取分页隧道列表缓存失败", "error", err, "username", username)
	}

	// 缓存未命中或出错，从数据库获取
	proxies, err := s.proxyRepo.GetByUsernameWithPagination(ctx, username, offset, limit)
	if err != nil {
		return nil, err
	}

	// 将结果存入缓存
	if len(proxies) > 0 {
		cacheBytes, err := json.Marshal(proxies)
		if err == nil {
			// 设置缓存，过期时间30分钟
			err = s.redisCli.Set(ctx, cacheKey, cacheBytes, 30*time.Minute).Err()
			if err != nil {
				s.logger.Error("设置分页隧道列表缓存失败", "error", err, "username", username)
			}
		}
	}

	return proxies, nil
}

// GetByUsernameAndName 根据用户名和隧道名称获取隧道
func (s *proxyService) GetByUsernameAndName(ctx context.Context, username, proxyName string) (*repository.Proxy, error) {
	return s.proxyRepo.GetByUsernameAndName(ctx, username, proxyName)
}

// Update 更新隧道
func (s *proxyService) Update(ctx context.Context, proxy *repository.Proxy) error {
	// 获取原隧道信息，用于后续可能的缓存处理
	oldProxy, err := s.proxyRepo.GetByID(ctx, proxy.ID)
	if err != nil {
		return err
	}

	err = s.proxyRepo.Update(ctx, proxy)
	if err != nil {
		return err
	}

	// 清除用户隧道缓存
	s.clearUserProxiesCache(ctx, proxy.Username)

	// 如果用户名发生了变更，也需要清除原用户的缓存
	if oldProxy != nil && oldProxy.Username != proxy.Username {
		s.clearUserProxiesCache(ctx, oldProxy.Username)
	}

	return nil
}

// Delete 删除隧道
func (s *proxyService) Delete(ctx context.Context, id int64) error {
	// 获取隧道信息，用于后续清除缓存
	proxy, err := s.proxyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.proxyRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 清除相关缓存
	if proxy != nil {
		s.clearUserProxiesCache(ctx, proxy.Username)
	}

	return nil
}

// List 获取隧道列表
func (s *proxyService) List(ctx context.Context, offset, limit int) ([]*repository.Proxy, error) {
	return s.proxyRepo.List(ctx, offset, limit)
}

// Count 获取隧道总数
func (s *proxyService) Count(ctx context.Context) (int, error) {
	return s.proxyRepo.Count(ctx)
}

// IsRemotePortUsed 检查同一节点下相同协议类型的隧道是否已经使用了相同的远程端口
func (s *proxyService) IsRemotePortUsed(ctx context.Context, nodeID int64, proxyType string, remotePort string) (bool, error) {
	return s.proxyRepo.IsRemotePortUsed(ctx, nodeID, proxyType, remotePort)
}

// GetUserProxyCount 获取用户的隧道数量
func (s *proxyService) GetUserProxyCount(ctx context.Context, username string) (int, error) {
	// 尝试从缓存获取
	cacheKey := s.getUserProxyCountCacheKey(username)
	count, err := s.redisCli.Get(ctx, cacheKey).Int()
	if err == nil {
		// 缓存命中，直接返回
		return count, nil
	} else if err.Error() != "redis: nil" {
		// 发生了除缓存不存在之外的错误
		s.logger.Error("获取用户隧道数量缓存失败", "error", err, "username", username)
	}

	// 缓存未命中或出错，从数据库获取
	proxies, err := s.proxyRepo.GetByUsername(ctx, username)
	if err != nil {
		return 0, err
	}

	count = len(proxies)

	// 将结果存入缓存
	err = s.redisCli.Set(ctx, cacheKey, count, 30*time.Minute).Err()
	if err != nil {
		s.logger.Error("设置用户隧道数量缓存失败", "error", err, "username", username)
	}

	return count, nil
}

// CheckUserNodeAccess 检查用户是否有权限使用特定节点
func (s *proxyService) CheckUserNodeAccess(ctx context.Context, username string, nodeID int64) (bool, error) {
	// 获取节点信息
	node, err := s.nodeService.GetByID(ctx, nodeID)
	if err != nil {
		return false, err
	}
	if node == nil {
		return false, errors.New("节点不存在")
	}

	// 获取用户信息
	user, err := s.userService.GetByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("用户不存在")
	}

	// 获取用户组信息
	group, err := s.userService.GetUserGroup(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if group == nil {
		return false, errors.New("用户组不存在")
	}

	// 使用工具函数检查用户组ID是否在节点的权限组列表中
	return utils.IsGroupInPermission(group.ID, node.Permission)
}
