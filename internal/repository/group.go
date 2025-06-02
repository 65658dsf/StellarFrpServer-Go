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
	ID                int64     `db:"id"`
	Name              string    `db:"name"`
	TunnelLimit       int       `db:"tunnel_limit"`
	BandwidthLimit    int       `db:"bandwidth_limit"`
	TrafficQuota      int64     `db:"traffic_quota"`
	CheckinMinTraffic int64     `db:"checkin_min_traffic"` // 签到最小流量(字节)
	CheckinMaxTraffic int64     `db:"checkin_max_traffic"` // 签到最大流量(字节)
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// GroupRepository 用户组仓库接口
type GroupRepository interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
	List(ctx context.Context) ([]*Group, error)
	Create(ctx context.Context, group *Group) error
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id int64) error
	SearchGroups(ctx context.Context, keyword string) ([]*Group, error)
	GetByName(ctx context.Context, name string) (*Group, error)
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

// GetByName 根据名称获取用户组
func (r *groupRepository) GetByName(ctx context.Context, name string) (*Group, error) {
	group := &Group{}
	query := `SELECT * FROM groups WHERE name = ?`
	err := r.db.GetContext(ctx, group, query, name)
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

// Create 创建用户组
func (r *groupRepository) Create(ctx context.Context, group *Group) error {
	query := `INSERT INTO groups (name, tunnel_limit, bandwidth_limit, traffic_quota, checkin_min_traffic, checkin_max_traffic) 
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		group.Name, group.TunnelLimit, group.BandwidthLimit, group.TrafficQuota,
		group.CheckinMinTraffic, group.CheckinMaxTraffic)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	group.ID = id
	return nil
}

// Update 更新用户组
func (r *groupRepository) Update(ctx context.Context, group *Group) error {
	query := `UPDATE groups SET name = ?, tunnel_limit = ?, bandwidth_limit = ?, traffic_quota = ?, 
		checkin_min_traffic = ?, checkin_max_traffic = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		group.Name, group.TunnelLimit, group.BandwidthLimit, group.TrafficQuota,
		group.CheckinMinTraffic, group.CheckinMaxTraffic, group.ID)
	return err
}

// Delete 删除用户组
func (r *groupRepository) Delete(ctx context.Context, id int64) error {
	// 检查是否有用户关联到此用户组
	var count int
	checkQuery := `SELECT COUNT(*) FROM users WHERE group_id = ?`
	err := r.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该用户组下存在用户，无法删除")
	}

	// 执行删除
	query := `DELETE FROM groups WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query, id)
	return err
}

// SearchGroups 搜索用户组
func (r *groupRepository) SearchGroups(ctx context.Context, keyword string) ([]*Group, error) {
	groups := []*Group{}
	query := `SELECT * FROM groups WHERE name LIKE ?`
	err := r.db.SelectContext(ctx, &groups, query, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	return groups, nil
}
