package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserCheckin 用户签到记录
type UserCheckin struct {
	ID            int64     `db:"id"`
	UserID        int64     `db:"user_id"`
	CheckinDate   time.Time `db:"checkin_date"`
	RewardTraffic int64     `db:"reward_traffic"`
	CreatedAt     time.Time `db:"created_at"`
}

// CheckinRepository 签到记录仓库接口
type CheckinRepository interface {
	Create(ctx context.Context, checkin *UserCheckin) error
	GetByUserAndDate(ctx context.Context, userID int64, date time.Time) (*UserCheckin, error)
	GetLatestByUser(ctx context.Context, userID int64) (*UserCheckin, error)
	GetUserCheckinCount(ctx context.Context, userID int64, startDate, endDate time.Time) (int, error)
}

// checkinRepository 签到记录仓库实现
type checkinRepository struct {
	db *sqlx.DB
}

// NewCheckinRepository 创建签到记录仓库实例
func NewCheckinRepository(db *sqlx.DB) CheckinRepository {
	return &checkinRepository{db: db}
}

// Create 创建签到记录
func (r *checkinRepository) Create(ctx context.Context, checkin *UserCheckin) error {
	query := `INSERT INTO user_checkin (user_id, checkin_date, reward_traffic, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	result, err := r.db.ExecContext(ctx, query,
		checkin.UserID, checkin.CheckinDate, checkin.RewardTraffic)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	checkin.ID = id
	return nil
}

// GetByUserAndDate 根据用户ID和日期获取签到记录
func (r *checkinRepository) GetByUserAndDate(ctx context.Context, userID int64, date time.Time) (*UserCheckin, error) {
	checkin := &UserCheckin{}
	query := `SELECT * FROM user_checkin WHERE user_id = ? AND DATE(checkin_date) = DATE(?)`
	err := r.db.GetContext(ctx, checkin, query, userID, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 返回nil表示没有签到记录
		}
		return nil, err
	}
	return checkin, nil
}

// GetLatestByUser 获取用户最新的签到记录
func (r *checkinRepository) GetLatestByUser(ctx context.Context, userID int64) (*UserCheckin, error) {
	checkin := &UserCheckin{}
	query := `SELECT * FROM user_checkin WHERE user_id = ? ORDER BY checkin_date DESC LIMIT 1`
	err := r.db.GetContext(ctx, checkin, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return checkin, nil
}

// GetUserCheckinCount 获取用户在指定日期范围内的签到次数
func (r *checkinRepository) GetUserCheckinCount(ctx context.Context, userID int64, startDate, endDate time.Time) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_checkin WHERE user_id = ? AND checkin_date BETWEEN ? AND ?`
	err := r.db.GetContext(ctx, &count, query, userID, startDate, endDate)
	if err != nil {
		return 0, err
	}
	return count, nil
}
