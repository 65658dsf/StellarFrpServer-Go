package service

import (
	"context"
	"errors"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/utils"
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
}

// NewProxyService 创建隧道服务实例
func NewProxyService(proxyRepo repository.ProxyRepository, nodeService NodeService, userService UserService) ProxyService {
	return &proxyService{
		proxyRepo:   proxyRepo,
		nodeService: nodeService,
		userService: userService,
	}
}

// Create 创建隧道
func (s *proxyService) Create(ctx context.Context, proxy *repository.Proxy) (int64, error) {
	return s.proxyRepo.Create(ctx, proxy)
}

// GetByID 根据ID获取隧道
func (s *proxyService) GetByID(ctx context.Context, id int64) (*repository.Proxy, error) {
	return s.proxyRepo.GetByID(ctx, id)
}

// GetByUsername 根据用户名获取隧道列表
func (s *proxyService) GetByUsername(ctx context.Context, username string) ([]*repository.Proxy, error) {
	return s.proxyRepo.GetByUsername(ctx, username)
}

// GetByUsernameWithPagination 根据用户名获取隧道列表（带分页）
func (s *proxyService) GetByUsernameWithPagination(ctx context.Context, username string, offset, limit int) ([]*repository.Proxy, error) {
	return s.proxyRepo.GetByUsernameWithPagination(ctx, username, offset, limit)
}

// GetByUsernameAndName 根据用户名和隧道名称获取隧道
func (s *proxyService) GetByUsernameAndName(ctx context.Context, username, proxyName string) (*repository.Proxy, error) {
	return s.proxyRepo.GetByUsernameAndName(ctx, username, proxyName)
}

// Update 更新隧道
func (s *proxyService) Update(ctx context.Context, proxy *repository.Proxy) error {
	return s.proxyRepo.Update(ctx, proxy)
}

// Delete 删除隧道
func (s *proxyService) Delete(ctx context.Context, id int64) error {
	return s.proxyRepo.Delete(ctx, id)
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
	proxies, err := s.proxyRepo.GetByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	return len(proxies), nil
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
