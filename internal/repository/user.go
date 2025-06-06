package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// User 用户模型
type User struct {
	ID                int64          `db:"id"`
	Username          string         `db:"username"`
	Password          string         `db:"password"`
	Email             string         `db:"email"`
	RegisterTime      time.Time      `db:"register_time"`
	GroupID           int64          `db:"group_id"`
	IsVerified        int            `db:"is_verified"`
	VerifyInfo        sql.NullString `db:"verify_info"`
	VerifyCount       int            `db:"verify_count"`
	Status            int            `db:"status"`
	Token             string         `db:"token"`
	GroupTime         *time.Time     `db:"group_time"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
	TunnelCount       *int           `db:"tunnel_count"`
	Bandwidth         *int           `db:"bandwidth"`
	TrafficQuota      *int64         `db:"traffic_quota"`
	LastCheckin       *time.Time     `db:"last_checkin"`       // 最后签到日期
	CheckinCount      int            `db:"checkin_count"`      // 签到总次数
	ContinuityCheckin int            `db:"continuity_checkin"` // 连续签到天数
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
	GetExpiredUsersByGroupID(ctx context.Context, groupID int64, expirationTime time.Time) ([]*User, error)
	Count(ctx context.Context) (int64, error)
	SearchUsers(ctx context.Context, keyword string) ([]*User, error)
}

// TransactionalUserRepository 扩展了UserRepository以支持事务
type TransactionalUserRepository interface {
	UserRepository
	BeginTx(ctx context.Context) (*sqlx.Tx, error)
	WithTx(tx *sqlx.Tx) UserRepository // 返回一个在事务上下文中操作的UserRepository
}

// userRepository 用户仓库实现
type userRepository struct {
	db *sqlx.DB // 直接数据库连接
	tx *sqlx.Tx // 可选的事务连接
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *sqlx.DB) TransactionalUserRepository { // 返回TransactionalUserRepository
	return &userRepository{db: db}
}

// BeginTx 开始一个新的事务
func (r *userRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

// WithTx 返回一个新的userRepository实例，该实例将在提供的事务上下文中操作
func (r *userRepository) WithTx(tx *sqlx.Tx) UserRepository {
	return &userRepository{db: r.db, tx: tx}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (username, password, email, register_time, group_id, is_verified, verify_info, verify_count, status, token, created_at, updated_at, tunnel_count, bandwidth, traffic_quota)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?)`
	var err error
	var result sql.Result
	if r.tx != nil {
		result, err = r.tx.ExecContext(ctx, query,
			user.Username, user.Password, user.Email,
			user.GroupID, user.IsVerified, user.VerifyInfo,
			user.VerifyCount, user.Status, user.Token,
			user.TunnelCount, user.Bandwidth, user.TrafficQuota)
	} else {
		result, err = r.db.ExecContext(ctx, query,
			user.Username, user.Password, user.Email,
			user.GroupID, user.IsVerified, user.VerifyInfo,
			user.VerifyCount, user.Status, user.Token,
			user.TunnelCount, user.Bandwidth, user.TrafficQuota)
	}
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
	var err error
	if r.tx != nil {
		err = r.tx.GetContext(ctx, user, query, id)
	} else {
		err = r.db.GetContext(ctx, user, query, id)
	}
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
	var err error
	if r.tx != nil {
		err = r.tx.GetContext(ctx, user, query, username)
	} else {
		err = r.db.GetContext(ctx, user, query, username)
	}
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
	var err error
	if r.tx != nil {
		err = r.tx.GetContext(ctx, user, query, email)
	} else {
		err = r.db.GetContext(ctx, user, query, email)
	}
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
		group_id = ?, is_verified = ?, verify_info = ?, verify_count = ?, status = ?, token = ?, group_time = ?,
		updated_at = CURRENT_TIMESTAMP, tunnel_count = ?, bandwidth = ?, traffic_quota = ?,
		last_checkin = ?, checkin_count = ?, continuity_checkin = ? WHERE id = ?`
	var err error
	if r.tx != nil {
		_, err = r.tx.ExecContext(ctx, query,
			user.Username, user.Password, user.Email, user.RegisterTime,
			user.GroupID, user.IsVerified, user.VerifyInfo, user.VerifyCount, user.Status, user.Token, user.GroupTime,
			user.TunnelCount, user.Bandwidth, user.TrafficQuota,
			user.LastCheckin, user.CheckinCount, user.ContinuityCheckin, user.ID)
	} else {
		_, err = r.db.ExecContext(ctx, query,
			user.Username, user.Password, user.Email, user.RegisterTime,
			user.GroupID, user.IsVerified, user.VerifyInfo, user.VerifyCount, user.Status, user.Token, user.GroupTime,
			user.TunnelCount, user.Bandwidth, user.TrafficQuota,
			user.LastCheckin, user.CheckinCount, user.ContinuityCheckin, user.ID)
	}
	return err
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	var err error
	if r.tx != nil {
		_, err = r.tx.ExecContext(ctx, query, id)
	} else {
		_, err = r.db.ExecContext(ctx, query, id)
	}
	return err
}

// List 获取用户列表
func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*User, error) {
	users := []*User{}
	query := `SELECT * FROM users LIMIT ? OFFSET ?`
	var err error
	if r.tx != nil {
		err = r.tx.SelectContext(ctx, &users, query, limit, offset)
	} else {
		err = r.db.SelectContext(ctx, &users, query, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetByToken 根据Token获取用户
func (r *userRepository) GetByToken(ctx context.Context, token string) (*User, error) {
	user := &User{}
	query := `SELECT * FROM users WHERE token = ?`
	var err error
	if r.tx != nil {
		err = r.tx.GetContext(ctx, user, query, token)
	} else {
		err = r.db.GetContext(ctx, user, query, token)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

// Count 获取用户总数
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM users`
	var err error
	if r.tx != nil {
		err = r.tx.GetContext(ctx, &count, query)
	} else {
		err = r.db.GetContext(ctx, &count, query)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SearchUsers 搜索用户
func (r *userRepository) SearchUsers(ctx context.Context, keyword string) ([]*User, error) {
	users := []*User{}
	// 使用LIKE查询搜索用户名、邮箱或ID
	query := `SELECT * FROM users WHERE username LIKE ? OR email LIKE ? OR id = ?`

	// 尝试将关键字转换为ID
	var id int64
	_, errScan := fmt.Sscanf(keyword, "%d", &id) // Renamed err to errScan to avoid conflict
	if errScan != nil {
		// 如果转换失败，使用0作为ID值（不会匹配任何记录）
		id = 0
	}

	// 在关键字前后添加%用于模糊匹配
	likeKeyword := "%" + keyword + "%"
	var err error
	if r.tx != nil {
		err = r.tx.SelectContext(ctx, &users, query, likeKeyword, likeKeyword, id)
	} else {
		err = r.db.SelectContext(ctx, &users, query, likeKeyword, likeKeyword, id)
	}
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetExpiredUsersByGroupID 根据 GroupID 和过期时间获取用户列表
func (r *userRepository) GetExpiredUsersByGroupID(ctx context.Context, groupID int64, expirationTime time.Time) ([]*User, error) {
	users := []*User{}
	query := `SELECT * FROM users WHERE group_id = ? AND group_time IS NOT NULL AND group_time < ?`
	var err error
	if r.tx != nil {
		err = r.tx.SelectContext(ctx, &users, query, groupID, expirationTime)
	} else {
		err = r.db.SelectContext(ctx, &users, query, groupID, expirationTime)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return users, nil // 没有找到匹配的用户，返回空列表而不是错误
		}
		return nil, err
	}
	return users, nil
}

// GetUserByID 根据ID获取用户
func (r *userRepository) GetUserByID(id uint64) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE id = ?`
	err := r.db.Get(&user, query, id)
	return &user, err
}

// UpdateUser 更新用户信息
func (r *userRepository) UpdateUser(user *User) error {
	query := `
		UPDATE users SET 
		username = ?, email = ?, password = ?, 
		verify_count = ?, status = ?,
		traffic_quota = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(
		query,
		user.Username,
		user.Email,
		user.Password,
		user.VerifyCount,
		user.Status,
		user.TrafficQuota,
		user.ID,
	)
	return err
}
