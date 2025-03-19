package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// Group 用户组模型
type Group struct {
	ID             int64     `db:"id"`
	Name           string    `db:"name"`
	TunnelLimit    int       `db:"tunnel_limit"`
	BandwidthLimit int       `db:"bandwidth_limit"`
	TrafficQuota   int64     `db:"traffic_quota"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// GroupRepository 用户组仓库接口
type GroupRepository interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
	List(ctx context.Context) ([]*Group, error)
}

// groupRepository 用户组仓库实现
type groupRepository struct {
	db *sqlx.DB
}

// NewGroupRepository 创建用户组仓库实例
func NewGroupRepository(db *sqlx.DB) GroupRepository {
	return &groupRepository{db: db}
}

// GetByID 根据ID获取用户组
func (r *groupRepository) GetByID(ctx context.Context, id int64) (*Group, error) {
	group := &Group{}
	query := `SELECT * FROM groups WHERE id = ?`
	err := r.db.GetContext(ctx, group, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group not found")
		}
		return nil, err
	}
	return group, nil
}

// List 获取所有用户组
func (r *groupRepository) List(ctx context.Context) ([]*Group, error) {
	groups := []*Group{}
	query := `SELECT * FROM groups`
	err := r.db.SelectContext(ctx, &groups, query)
	if err != nil {
		return nil, err
	}
	return groups, nil
}
