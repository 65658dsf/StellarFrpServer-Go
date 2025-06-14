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
	GetNodesByOwnerID(ctx context.Context, ownerID int64) ([]*repository.Node, error)
	List(ctx context.Context, offset, limit int) ([]*repository.Node, error)
	GetAllNodes(ctx context.Context) ([]*repository.Node, error)
	CreateNode(ctx context.Context, node *repository.Node) error
	GetLatestNodeTraffic(ctx context.Context, nodeName string) (*repository.NodeTrafficLog, error)
	GetTotalTraffic(ctx context.Context) (int64, int64, error) // 返回总入站流量和总出站流量
}

// nodeService 节点服务实现
type nodeService struct {
	nodeRepo        repository.NodeRepository
	nodeTrafficRepo repository.NodeTrafficRepository
}

// NewNodeService 创建节点服务实例
func NewNodeService(nodeRepo repository.NodeRepository, nodeTrafficRepo repository.NodeTrafficRepository) NodeService {
	return &nodeService{
		nodeRepo:        nodeRepo,
		nodeTrafficRepo: nodeTrafficRepo,
	}
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

// GetNodesByOwnerID 根据所属用户ID获取节点列表
func (s *nodeService) GetNodesByOwnerID(ctx context.Context, ownerID int64) ([]*repository.Node, error) {
	return s.nodeRepo.GetByOwnerID(ctx, ownerID)
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

// CreateNode 创建节点
func (s *nodeService) CreateNode(ctx context.Context, node *repository.Node) error {
	return s.nodeRepo.Create(ctx, node)
}

// GetLatestNodeTraffic 获取指定节点的最新流量记录
func (s *nodeService) GetLatestNodeTraffic(ctx context.Context, nodeName string) (*repository.NodeTrafficLog, error) {
	// 首先检查节点是否存在
	_, err := s.nodeRepo.GetByNodeName(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	// 获取该节点的最新流量记录
	return s.nodeTrafficRepo.GetLastRecord(ctx, nodeName)
}

// GetTotalTraffic 获取所有节点的总流量
func (s *nodeService) GetTotalTraffic(ctx context.Context) (int64, int64, error) {
	// 直接从流量记录表中获取所有流量的总和
	return s.nodeTrafficRepo.GetTotalTraffic(ctx)
}
