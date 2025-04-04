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
	IsIncrement bool      `db:"is_increment"`
}

// NodeTrafficRepository 节点流量仓库接口
type NodeTrafficRepository interface {
	Create(ctx context.Context, traffic *NodeTrafficLog) error
	GetLastRecord(ctx context.Context, nodeName string) (*NodeTrafficLog, error)
	GetTodayIncrement(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error)
	GetTodayTotal(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error)
	UpdateIncrement(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error
	UpdateTotal(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error
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
	query := `INSERT INTO node_traffic_log (node_name, traffic_in, traffic_out, online_count, record_time, record_date, is_increment) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		traffic.NodeName, traffic.TrafficIn, traffic.TrafficOut, traffic.OnlineCount,
		traffic.RecordTime, traffic.RecordDate, traffic.IsIncrement)
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

// GetTodayIncrement 获取指定节点当天的增量流量记录
func (r *nodeTrafficRepository) GetTodayIncrement(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error) {
	query := `SELECT * FROM node_traffic_log WHERE node_name = ? AND record_date = ? AND is_increment = TRUE LIMIT 1`
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

// GetTodayTotal 获取指定节点当天的总流量记录
func (r *nodeTrafficRepository) GetTodayTotal(ctx context.Context, nodeName string, date string) (*NodeTrafficLog, error) {
	query := `SELECT * FROM node_traffic_log WHERE node_name = ? AND record_date = ? AND is_increment = FALSE LIMIT 1`
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

// UpdateIncrement 更新增量流量记录
func (r *nodeTrafficRepository) UpdateIncrement(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error {
	query := `UPDATE node_traffic_log SET traffic_in = traffic_in + ?, traffic_out = traffic_out + ?, online_count = ?, record_time = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, trafficIn, trafficOut, onlineCount, id)
	return err
}

// UpdateTotal 更新总流量记录
func (r *nodeTrafficRepository) UpdateTotal(ctx context.Context, id int64, trafficIn, trafficOut int64, onlineCount int) error {
	query := `UPDATE node_traffic_log SET traffic_in = ?, traffic_out = ?, online_count = ?, record_time = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, trafficIn, trafficOut, onlineCount, id)
	return err
}
