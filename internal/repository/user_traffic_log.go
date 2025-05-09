package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"stellarfrp/internal/types"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserTrafficLogRepository 用户流量日志仓库接口
type UserTrafficLogRepository interface {
	// 确保表存在
	EnsureTableExists(ctx context.Context) error
	// 获取用户流量记录
	GetByUsername(ctx context.Context, username string) (*types.UserTrafficLog, error)
	// 获取所有用户的流量记录
	GetAll(ctx context.Context) ([]*types.UserTrafficLog, error)
	// 更新用户流量记录
	UpdateTraffic(ctx context.Context, username string, todayTraffic int64) error
}

// userTrafficLogRepository 用户流量日志仓库实现
type userTrafficLogRepository struct {
	db *sqlx.DB
}

// NewUserTrafficLogRepository 创建用户流量日志仓库实例
func NewUserTrafficLogRepository(db *sqlx.DB) UserTrafficLogRepository {
	return &userTrafficLogRepository{db: db}
}

// EnsureTableExists 确保user_traffic_log表存在
func (r *userTrafficLogRepository) EnsureTableExists(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS user_traffic_log (
		id BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
		username VARCHAR(100) NOT NULL,
		total_traffic BIGINT(20) NOT NULL DEFAULT 0,
		today_traffic BIGINT(20) NOT NULL DEFAULT 0,
		history_traffic TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY (username)
	)`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

// GetByUsername 根据用户名获取流量记录
func (r *userTrafficLogRepository) GetByUsername(ctx context.Context, username string) (*types.UserTrafficLog, error) {
	log := &types.UserTrafficLog{}
	query := `SELECT * FROM user_traffic_log WHERE username = ?`
	err := r.db.GetContext(ctx, log, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 返回nil表示没有找到记录，但不是错误
		}
		return nil, err
	}
	return log, nil
}

// GetAll 获取所有用户的流量记录
func (r *userTrafficLogRepository) GetAll(ctx context.Context) ([]*types.UserTrafficLog, error) {
	logs := []*types.UserTrafficLog{}
	query := `SELECT * FROM user_traffic_log`
	err := r.db.SelectContext(ctx, &logs, query)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// UpdateTraffic 更新用户流量记录
// 如果用户记录不存在则创建新记录
func (r *userTrafficLogRepository) UpdateTraffic(ctx context.Context, username string, todayTraffic int64) error {
	// 获取现有记录
	existingLog, err := r.GetByUsername(ctx, username)
	if err != nil {
		return err
	}

	// 开始事务
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if existingLog == nil {
		// 如果记录不存在，创建新记录
		historyTraffic := []int64{todayTraffic}
		historyTrafficJSON, err := json.Marshal(historyTraffic)
		if err != nil {
			return err
		}

		query := `INSERT INTO user_traffic_log 
		(username, total_traffic, today_traffic, history_traffic) 
		VALUES (?, ?, ?, ?)`
		_, err = tx.ExecContext(ctx, query, username, todayTraffic, todayTraffic, string(historyTrafficJSON))
		if err != nil {
			return err
		}
	} else {
		// 如果记录存在，更新记录
		var historyTraffic []int64
		if err := json.Unmarshal([]byte(existingLog.HistoryTraffic), &historyTraffic); err != nil {
			historyTraffic = []int64{}
		}

		// 检查是否是同一天
		sameDay := isSameDay(existingLog.UpdatedAt)
		var totalTraffic int64

		if sameDay {
			// 同一天，计算增量
			increment := todayTraffic - existingLog.TodayTraffic
			if increment < 0 {
				increment = 0 // 如果是负数，则设为0
			}
			totalTraffic = existingLog.TotalTraffic + increment
			// 更新历史记录
			if len(historyTraffic) > 0 {
				historyTraffic[0] = todayTraffic
			} else {
				historyTraffic = append(historyTraffic, todayTraffic)
			}
		} else {
			// 不是同一天，添加新的记录
			totalTraffic = existingLog.TotalTraffic + todayTraffic
			historyTraffic = append([]int64{todayTraffic}, historyTraffic...)
		}

		// 限制历史记录最多30天
		if len(historyTraffic) > 30 {
			historyTraffic = historyTraffic[:30]
		}

		historyTrafficJSON, err := json.Marshal(historyTraffic)
		if err != nil {
			return err
		}

		query := `UPDATE user_traffic_log 
		SET total_traffic = ?, today_traffic = ?, history_traffic = ?
		WHERE id = ?`
		_, err = tx.ExecContext(ctx, query, totalTraffic, todayTraffic, string(historyTrafficJSON), existingLog.ID)
		if err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
}

// isSameDay 判断时间是否为今天
func isSameDay(t time.Time) bool {
	now := time.Now()
	return now.Year() == t.Year() && now.Month() == t.Month() && now.Day() == t.Day()
}
