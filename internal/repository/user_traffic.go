package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// HistoryTraffic 历史流量记录
type HistoryTraffic struct {
	Date    string `json:"date"`
	Traffic int64  `json:"traffic"`
}

// UserTrafficLog 用户流量日志模型
type UserTrafficLog struct {
	ID             int64          `db:"id"`
	Username       string         `db:"username"`
	TotalTraffic   int64          `db:"total_traffic"`
	TodayTraffic   int64          `db:"today_traffic"`
	HistoryTraffic sql.NullString `db:"history_traffic"` // JSON格式的历史流量记录
	TrafficQuota   int64          `db:"traffic_quota"`   // 用户总流量配额
	UsagePercent   float64        `db:"usage_percent"`   // 已使用流量百分比
	RecordDate     string         `db:"record_date"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

// UserTrafficRepository 用户流量仓库接口
type UserTrafficRepository interface {
	// 获取指定用户当天的流量记录
	GetByUsername(ctx context.Context, username string, date string) (*UserTrafficLog, error)

	// 获取所有用户当天的流量记录
	GetAllUserTraffic(ctx context.Context, date string) ([]*UserTrafficLog, error)

	// 创建或更新用户流量记录
	CreateOrUpdate(ctx context.Context, traffic *UserTrafficLog) error

	// 更新历史流量记录并保持最多14天
	UpdateHistoryTraffic(ctx context.Context, username string, date string, todayTraffic int64) error
}

// userTrafficRepository 用户流量仓库实现
type userTrafficRepository struct {
	db *sqlx.DB
}

// NewUserTrafficRepository 创建用户流量仓库实例
func NewUserTrafficRepository(db *sqlx.DB) UserTrafficRepository {
	return &userTrafficRepository{db: db}
}

// GetByUsername 获取指定用户当天的流量记录
func (r *userTrafficRepository) GetByUsername(ctx context.Context, username string, date string) (*UserTrafficLog, error) {
	query := `SELECT * FROM user_traffic_log WHERE username = ? AND record_date = ? LIMIT 1`
	traffic := &UserTrafficLog{}
	err := r.db.GetContext(ctx, traffic, query, username, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return traffic, nil
}

// GetAllUserTraffic 获取所有用户当天的流量记录
func (r *userTrafficRepository) GetAllUserTraffic(ctx context.Context, date string) ([]*UserTrafficLog, error) {
	query := `SELECT * FROM user_traffic_log WHERE record_date = ?`
	var records []*UserTrafficLog
	err := r.db.SelectContext(ctx, &records, query, date)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// CreateOrUpdate 创建或更新用户流量记录
func (r *userTrafficRepository) CreateOrUpdate(ctx context.Context, traffic *UserTrafficLog) error {
	query := `
		INSERT INTO user_traffic_log
		(username, total_traffic, today_traffic, history_traffic, traffic_quota, usage_percent, record_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		total_traffic = VALUES(total_traffic),
		today_traffic = VALUES(today_traffic),
		history_traffic = VALUES(history_traffic),
		traffic_quota = VALUES(traffic_quota),
		usage_percent = VALUES(usage_percent)
	`
	_, err := r.db.ExecContext(ctx, query,
		traffic.Username,
		traffic.TotalTraffic,
		traffic.TodayTraffic,
		traffic.HistoryTraffic,
		traffic.TrafficQuota,
		traffic.UsagePercent,
		traffic.RecordDate,
	)
	return err
}

// UpdateHistoryTraffic 更新历史流量记录并保持最多14天
func (r *userTrafficRepository) UpdateHistoryTraffic(ctx context.Context, username string, date string, todayTraffic int64) error {
	// 获取当前用户的流量记录
	record, err := r.GetByUsername(ctx, username, date)
	if err != nil {
		return err
	}

	// 解析历史流量记录
	var historyTraffic []HistoryTraffic
	if record != nil && record.HistoryTraffic.Valid {
		err = json.Unmarshal([]byte(record.HistoryTraffic.String), &historyTraffic)
		if err != nil {
			historyTraffic = []HistoryTraffic{}
		}
	} else {
		historyTraffic = []HistoryTraffic{}
	}

	// 添加今天的流量记录到历史记录中
	yesterdayDate := time.Now().Add(-24 * time.Hour).Format("2006/01/02")
	newEntry := HistoryTraffic{
		Date:    yesterdayDate,
		Traffic: todayTraffic,
	}

	// 检查是否已存在相同日期的记录，如果有则更新
	exists := false
	for i, entry := range historyTraffic {
		if entry.Date == yesterdayDate {
			historyTraffic[i].Traffic = todayTraffic
			exists = true
			break
		}
	}

	// 如果不存在则添加
	if !exists {
		historyTraffic = append(historyTraffic, newEntry)
	}

	// 只保留最近14天的记录
	if len(historyTraffic) > 14 {
		historyTraffic = historyTraffic[len(historyTraffic)-14:]
	}

	// 更新数据库
	historyJSON, err := json.Marshal(historyTraffic)
	if err != nil {
		return err
	}

	// 如果记录不存在，则创建一个新记录
	if record == nil {
		record = &UserTrafficLog{
			Username:     username,
			TotalTraffic: todayTraffic,
			TodayTraffic: 0, // 新的一天，今日流量重置为0
			HistoryTraffic: sql.NullString{
				String: string(historyJSON),
				Valid:  true,
			},
			RecordDate: date,
		}
	} else {
		// 更新现有记录
		totalTraffic := record.TotalTraffic
		if record.RecordDate != date {
			// 如果是新的一天，则今日流量重置为0，但总流量保持不变
			record.TodayTraffic = 0
		}
		record.HistoryTraffic = sql.NullString{
			String: string(historyJSON),
			Valid:  true,
		}
		record.TotalTraffic = totalTraffic
	}

	return r.CreateOrUpdate(ctx, record)
}
