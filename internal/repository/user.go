package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// User 用户模型
type User struct {
	ID           int64      `db:"id"`
	Username     string     `db:"username"`
	Password     string     `db:"password"`
	Email        string     `db:"email"`
	RegisterTime time.Time  `db:"register_time"`
	GroupID      int64      `db:"group_id"`
	IsVerified   int        `db:"is_verified"`
	VerifyInfo   string     `db:"verify_info"`
	VerifyCount  int        `db:"verify_count"`
	Status       int        `db:"status"`
	Token        string     `db:"token"`
	GroupTime    *time.Time `db:"group_time"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	TunnelCount  *int       `db:"tunnel_count"`
	Bandwidth    *int       `db:"bandwidth"`
	TrafficQuota *int64     `db:"traffic_quota"`
}

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByToken(ctx context.Context, token string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*User, error)
}

// userRepository 用户仓库实现
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (username, password, email, register_time, group_id, is_verified, verify_info, verify_count, status, token, created_at, updated_at, tunnel_count, bandwidth, traffic_quota) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		user.Username, user.Password, user.Email,
		user.GroupID, user.IsVerified, user.VerifyInfo,
		user.VerifyCount, user.Status, user.Token,
		user.TunnelCount, user.Bandwidth, user.TrafficQuota)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	query := `SELECT * FROM users WHERE id = ?`
	err := r.db.GetContext(ctx, user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	query := `SELECT * FROM users WHERE username = ?`
	err := r.db.GetContext(ctx, user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `SELECT * FROM users WHERE email = ?`
	err := r.db.GetContext(ctx, user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return user, nil
}

// Update 更新用户信息
func (r *userRepository) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET username = ?, password = ?, email = ?, register_time = ?, 
		group_id = ?, is_verified = ?, verify_info = ?, verify_count = ?, status = ?, token = ?, group_time = ?, updated_at = CURRENT_TIMESTAMP, tunnel_count = ?, bandwidth = ?, traffic_quota = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		user.Username, user.Password, user.Email, user.RegisterTime,
		user.GroupID, user.IsVerified, user.VerifyInfo, user.VerifyCount, user.Status, user.Token, user.GroupTime,
		user.TunnelCount, user.Bandwidth, user.TrafficQuota, user.ID)
	return err
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List 获取用户列表
func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*User, error) {
	users := []*User{}
	query := `SELECT * FROM users LIMIT ? OFFSET ?`
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetByToken 根据Token获取用户
func (r *userRepository) GetByToken(ctx context.Context, token string) (*User, error) {
	user := &User{}
	query := `SELECT * FROM users WHERE token = ?`
	err := r.db.GetContext(ctx, user, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}
