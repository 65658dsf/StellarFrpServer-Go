package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// NodeTrafficLog 节点流量日志模型
type NodeTrafficLog struct {
	ID          int64     `db:"id"`
	NodeName    string    `db:"node_name"`
	TrafficIn   int64     `db:"traffic_in"`
	TrafficOut  int64     `db:"traffic_out"`
	OnlineCount int       `db:"online_count"`
	RecordTime  time.Time `db:"record_time"`
	RecordDate  string    `db:"record_date"`
}

// NodeTrafficRepository 节点流量仓库接口
type NodeTrafficRepository interface {
	Create(ctx context.Context, traffic *NodeTrafficLog) error
	GetLastRecord(ctx context.Context, nodeName string) (*NodeTrafficLog, error)
	GetDailyRecord(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error)
	UpdateRecord(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error
	GetTotalTraffic(ctx context.Context) (int64, int64, error)
}

// nodeTrafficRepository 节点流量仓库实现
type nodeTrafficRepository struct {
	db *sqlx.DB
}

// NewNodeTrafficRepository 创建节点流量仓库实例
func NewNodeTrafficRepository(db *sqlx.DB) NodeTrafficRepository {
	return &nodeTrafficRepository{db: db}
}

// Create 创建节点流量记录
func (r *nodeTrafficRepository) Create(ctx context.Context, traffic *NodeTrafficLog) error {
	query := `INSERT INTO node_traffic_log (node_name, traffic_in, traffic_out, online_count, record_time, record_date) 
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		traffic.NodeName, traffic.TrafficIn, traffic.TrafficOut, traffic.OnlineCount,
		traffic.RecordTime, traffic.RecordDate)
	return err
}

// GetLastRecord 获取指定节点的最新流量记录
func (r *nodeTrafficRepository) GetLastRecord(ctx context.Context, nodeName string) (*NodeTrafficLog, error) {
	query := `SELECT * FROM node_traffic_log WHERE node_name = ? ORDER BY record_time DESC LIMIT 1`
	traffic := &NodeTrafficLog{}
	err := r.db.GetContext(ctx, traffic, query, nodeName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return traffic, nil
}

// GetDailyRecord 获取指定节点指定日期的流量记录
func (r *nodeTrafficRepository) GetDailyRecord(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error) {
	query := `SELECT * FROM node_traffic_log WHERE node_name = ? AND record_date = ? LIMIT 1`
	traffic := &NodeTrafficLog{}
	err := r.db.GetContext(ctx, traffic, query, nodeName, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return traffic, nil
}

// UpdateRecord 更新流量记录
func (r *nodeTrafficRepository) UpdateRecord(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error {
	query := `UPDATE node_traffic_log SET traffic_in = ?, traffic_out = ?, online_count = ?, record_time = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, trafficIn, trafficOut, onlineCount, id)
	return err
}

// 以下是为了兼容旧代码，保留的方法
// GetTodayIncrement 获取指定节点当天的流量记录 (兼容旧方法)
func (r *nodeTrafficRepository) GetTodayIncrement(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error) {
	return r.GetDailyRecord(ctx, nodeName, date)
}

// GetTodayTotal 获取指定节点当天的流量记录 (兼容旧方法)
func (r *nodeTrafficRepository) GetTodayTotal(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error) {
	return r.GetDailyRecord(ctx, nodeName, date)
}

// UpdateIncrement 更新流量记录 (兼容旧方法)
func (r *nodeTrafficRepository) UpdateIncrement(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error {
	// 不再使用增量更新，而是直接设置值
	return r.UpdateRecord(ctx, id, trafficIn, trafficOut, onlineCount)
}

// UpdateTotal 更新流量记录 (兼容旧方法)
func (r *nodeTrafficRepository) UpdateTotal(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error {
	return r.UpdateRecord(ctx, id, trafficIn, trafficOut, onlineCount)
}

// GetTotalTraffic 获取所有流量记录的总和
func (r *nodeTrafficRepository) GetTotalTraffic(ctx context.Context) (int64, int64, error) {
	var result struct {
		TotalIn  int64 `db:"total_in"`
		TotalOut int64 `db:"total_out"`
	}

	// 直接从流量记录表中查询所有流量的总和
	query := `
		SELECT 
			COALESCE(SUM(traffic_in), 0) AS total_in,
			COALESCE(SUM(traffic_out), 0) AS total_out
		FROM node_traffic_log
	`

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return 0, 0, err
	}

	return result.TotalIn, result.TotalOut, nil
}
