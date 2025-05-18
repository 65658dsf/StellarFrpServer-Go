package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserCheckinLog 用户签到记录模型
type UserCheckinLog struct {
	ID             int64     `db:"id"`
	UserID         int64     `db:"user_id"`
	Username       string    `db:"username"`
	CheckinDate    time.Time `db:"checkin_date"`
	RewardTraffic  int64     `db:"reward_traffic"`
	ContinuityDays int       `db:"continuity_days"`
	CreatedAt      time.Time `db:"created_at"`
}

// UserCheckinRepository 用户签到记录仓库接口
type UserCheckinRepository interface {
	// 创建签到记录
	Create(ctx context.Context, checkin *UserCheckinLog) error

	// 获取用户的签到记录及总数
	GetByUserIDWithTotal(ctx context.Context, userID int64, limit, offset int) ([]*UserCheckinLog, int, error)

	// 获取用户最近的签到记录
	GetLatestByUserID(ctx context.Context, userID int64) (*UserCheckinLog, error)

	// 获取用户特定日期的签到记录
	GetByUserAndDate(ctx context.Context, userID int64, date time.Time) (*UserCheckinLog, error)

	// 获取今日签到用户数
	GetTodayCheckinCount(ctx context.Context) (int, error)
}

// userCheckinRepository 用户签到记录仓库实现
type userCheckinRepository struct {
	db *sqlx.DB
}

// NewUserCheckinRepository 创建用户签到记录仓库实例
func NewUserCheckinRepository(db *sqlx.DB) UserCheckinRepository {
	return &userCheckinRepository{db: db}
}

// Create 创建签到记录
func (r *userCheckinRepository) Create(ctx context.Context, checkin *UserCheckinLog) error {
	query := `INSERT INTO user_checkin_logs (user_id, username, checkin_date, reward_traffic, continuity_days) 
		VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		checkin.UserID, checkin.Username, checkin.CheckinDate, checkin.RewardTraffic, checkin.ContinuityDays)
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

// GetLatestByUserID 获取用户最近的签到记录
func (r *userCheckinRepository) GetLatestByUserID(ctx context.Context, userID int64) (*UserCheckinLog, error) {
	query := `SELECT * FROM user_checkin_logs WHERE user_id = ? ORDER BY checkin_date DESC LIMIT 1`
	log := &UserCheckinLog{}
	err := r.db.GetContext(ctx, log, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// GetByUserAndDate 获取用户特定日期的签到记录
func (r *userCheckinRepository) GetByUserAndDate(ctx context.Context, userID int64, date time.Time) (*UserCheckinLog, error) {
	// 格式化日期为YYYY-MM-DD格式，忽略时间部分
	dateStr := date.Format("2006-01-02")
	query := `SELECT * FROM user_checkin_logs WHERE user_id = ? AND DATE(checkin_date) = ? LIMIT 1`
	log := &UserCheckinLog{}
	err := r.db.GetContext(ctx, log, query, userID, dateStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// GetTodayCheckinCount 获取今日签到用户数
func (r *userCheckinRepository) GetTodayCheckinCount(ctx context.Context) (int, error) {
	// 获取今天的日期（YYYY-MM-DD格式）
	today := time.Now().Format("2006-01-02")
	query := `SELECT COUNT(*) FROM user_checkin_logs WHERE DATE(checkin_date) = ?`
	var count int
	err := r.db.GetContext(ctx, &count, query, today)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetByUserIDWithTotal 获取用户的签到记录及总数
func (r *userCheckinRepository) GetByUserIDWithTotal(ctx context.Context, userID int64, limit, offset int) ([]*UserCheckinLog, int, error) {
	// 先获取总记录数
	countQuery := `SELECT COUNT(*) FROM user_checkin_logs WHERE user_id = ?`
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有记录，直接返回空数组和0
	if total == 0 {
		return []*UserCheckinLog{}, 0, nil
	}

	// 获取分页数据
	dataQuery := `SELECT * FROM user_checkin_logs WHERE user_id = ? ORDER BY checkin_date DESC LIMIT ? OFFSET ?`
	var logs []*UserCheckinLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
