package service

import (
	"context"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
)

// GroupService 用户组服务接口
type GroupService interface {
	GetByID(ctx context.Context, id int64) (*repository.Group, error)
	List(ctx context.Context) ([]*repository.Group, error)
	Create(ctx context.Context, group *repository.Group) error
	Update(ctx context.Context, group *repository.Group) error
	Delete(ctx context.Context, id int64) error
	SearchGroups(ctx context.Context, keyword string) ([]*repository.Group, error)
	GetByName(ctx context.Context, name string) (*repository.Group, error)
}

// groupService 用户组服务实现
type groupService struct {
	groupRepo repository.GroupRepository
	logger    *logger.Logger
}

// NewGroupService 创建用户组服务实例
func NewGroupService(groupRepo repository.GroupRepository, logger *logger.Logger) GroupService {
	return &groupService{
		groupRepo: groupRepo,
		logger:    logger,
	}
}

// GetByID 根据ID获取用户组
func (s *groupService) GetByID(ctx context.Context, id int64) (*repository.Group, error) {
	return s.groupRepo.GetByID(ctx, id)
}

// GetByName 根据名称获取用户组
func (s *groupService) GetByName(ctx context.Context, name string) (*repository.Group, error) {
	return s.groupRepo.GetByName(ctx, name)
}

// List 获取所有用户组
func (s *groupService) List(ctx context.Context) ([]*repository.Group, error) {
	return s.groupRepo.List(ctx)
}

// Create 创建用户组
func (s *groupService) Create(ctx context.Context, group *repository.Group) error {
	return s.groupRepo.Create(ctx, group)
}

// Update 更新用户组
func (s *groupService) Update(ctx context.Context, group *repository.Group) error {
	return s.groupRepo.Update(ctx, group)
}

// Delete 删除用户组
func (s *groupService) Delete(ctx context.Context, id int64) error {
	return s.groupRepo.Delete(ctx, id)
}

// SearchGroups 搜索用户组
func (s *groupService) SearchGroups(ctx context.Context, keyword string) ([]*repository.Group, error) {
	return s.groupRepo.SearchGroups(ctx, keyword)
}
