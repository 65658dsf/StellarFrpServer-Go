package service

import (
	"context"
	"stellarfrp/internal/repository"
)

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, user *repository.User) error
	CreateAsync(ctx context.Context, user *repository.User) (string, error)
	GetByID(ctx context.Context, id int64) (*repository.User, error)
	GetByUsername(ctx context.Context, username string) (*repository.User, error)
	GetByEmail(ctx context.Context, email string) (*repository.User, error)
	GetByToken(ctx context.Context, token string) (*repository.User, error)
	Update(ctx context.Context, user *repository.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]*repository.User, error)
	GetAllUsers(ctx context.Context) ([]*repository.User, error)
	GetUsersWithExpiredGroup(ctx context.Context, groupID int64) ([]*repository.User, error)
	Count(ctx context.Context) (int64, error)
	GetTaskStatus(ctx context.Context, taskID string) (string, error)
	SendEmail(ctx context.Context, email, msgType string) error
	Login(ctx context.Context, identifier, password string) (*repository.User, error)
	GetGroupName(ctx context.Context, groupID int64) (string, error)
	GetGroupTunnelLimit(ctx context.Context, groupID int64) (int, error)
	GetUserBandwidth(ctx context.Context, userID int64) (int, error)
	GetUserTrafficQuota(ctx context.Context, userID int64) (int64, error)
	GetUserGroup(ctx context.Context, userID int64) (*repository.Group, error)
	ResetToken(ctx context.Context, identifier, password string) (*repository.User, error)
	AdminResetToken(ctx context.Context, user *repository.User) error
	GetGroupTraffic(ctx context.Context, groupID int64) (int64, error)
	GetUserUsedTraffic(ctx context.Context, userID int64) (int64, error)
	IsUserBlacklisted(ctx context.Context, userID int64) (bool, error)
	IsUserBlacklistedByUsername(ctx context.Context, username string) (bool, error)
	IsUserBlacklistedByToken(ctx context.Context, token string) (bool, error)
	SearchUsers(ctx context.Context, keyword string) ([]*repository.User, error)
	GetAllGroups(ctx context.Context) ([]*repository.Group, error)

	// 新增方法
	AddVerifyCount(userID uint64, count int) error
	AddTraffic(userID uint64, trafficGB float64) error
	UpdateUserGroup(ctx context.Context, userID int64, groupID int64) error
}
