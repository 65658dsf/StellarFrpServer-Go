package repository

import (
	"context"
	"fmt"
	"stellarfrp/internal/model"

	"github.com/jmoiron/sqlx"
)

// SystemRepository 系统状态存储库
type SystemRepository struct {
	db *sqlx.DB
}

// NewSystemRepository 创建系统状态存储库实例
func NewSystemRepository(db *sqlx.DB) *SystemRepository {
	return &SystemRepository{db: db}
}

// GetSystemStatus 获取系统状态
func (r *SystemRepository) GetSystemStatus(ctx context.Context) (*model.SystemStatus, error) {
	var status model.SystemStatus

	// 获取用户总数
	err := r.db.GetContext(ctx, &status.TotalUsers, "SELECT COUNT(*) FROM users")
	if err != nil {
		return nil, err
	}

	// 获取隧道总数
	err = r.db.GetContext(ctx, &status.TotalTunnels, "SELECT COUNT(*) FROM proxy")
	if err != nil {
		return nil, err
	}

	// 获取节点总数
	err = r.db.GetContext(ctx, &status.TotalNodes, "SELECT COUNT(*) FROM nodes")
	if err != nil {
		return nil, err
	}

	// 获取总流量
	var traffic struct {
		TotalIn  int64 `db:"total_in"`
		TotalOut int64 `db:"total_out"`
	}
	err = r.db.GetContext(ctx, &traffic, `
		SELECT 
			COALESCE(SUM(traffic_in), 0) as total_in,
			COALESCE(SUM(traffic_out), 0) as total_out
		FROM node_traffic_log
	`)
	if err != nil {
		return nil, err
	}

	// 计算总流量并格式化
	totalBytes := traffic.TotalIn + traffic.TotalOut
	status.TotalTraffic = formatTraffic(totalBytes)

	return &status, nil
}

// formatTraffic 格式化流量
func formatTraffic(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	trafficValue := float64(bytes)

	for trafficValue >= 1024 && unitIndex < len(units)-1 {
		trafficValue /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f%s", trafficValue, units[unitIndex])
}
