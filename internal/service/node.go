package service

import (
	"context"
	"stellarfrp/internal/repository"
)

// NodeService 节点服务接口
type NodeService interface {
	GetByID(ctx context.Context, id int64) (*repository.Node, error)
	GetByNodeName(ctx context.Context, nodeName string) (*repository.Node, error)
	GetByUser(ctx context.Context, user string) ([]*repository.Node, error)
	GetAccessibleNodes(ctx context.Context, groupID int64) ([]*repository.Node, error)
	List(ctx context.Context, offset, limit int) ([]*repository.Node, error)
	GetAllNodes(ctx context.Context) ([]*repository.Node, error)
}

// nodeService 节点服务实现
type nodeService struct {
	nodeRepo repository.NodeRepository
}

// NewNodeService 创建节点服务实例
func NewNodeService(nodeRepo repository.NodeRepository) NodeService {
	return &nodeService{nodeRepo: nodeRepo}
}

// GetByID 根据ID获取节点
func (s *nodeService) GetByID(ctx context.Context, id int64) (*repository.Node, error) {
	return s.nodeRepo.GetByID(ctx, id)
}

// GetByNodeName 根据节点名称获取节点
func (s *nodeService) GetByNodeName(ctx context.Context, nodeName string) (*repository.Node, error) {
	return s.nodeRepo.GetByNodeName(ctx, nodeName)
}

// GetByUser 根据用户获取节点列表
func (s *nodeService) GetByUser(ctx context.Context, user string) ([]*repository.Node, error) {
	return s.nodeRepo.GetByUser(ctx, user)
}

// GetAccessibleNodes 获取用户组可访问的所有节点
// 包括公共节点(permission=[]或空)和权限数组中包含用户组ID的节点
func (s *nodeService) GetAccessibleNodes(ctx context.Context, groupID int64) ([]*repository.Node, error) {
	return s.nodeRepo.GetByPermission(ctx, groupID)
}

// List 获取节点列表
func (s *nodeService) List(ctx context.Context, offset, limit int) ([]*repository.Node, error) {
	return s.nodeRepo.List(ctx, offset, limit)
}

// GetAllNodes 获取所有节点
func (s *nodeService) GetAllNodes(ctx context.Context) ([]*repository.Node, error) {
	// 使用List方法获取所有节点，不分页
	return s.nodeRepo.List(ctx, 0, 10000) // 设置一个足够大的数字
}
