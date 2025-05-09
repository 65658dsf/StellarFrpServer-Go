package types

import "time"

// UserTrafficLog 用户流量日志结构
type UserTrafficLog struct {
	ID             int64     `db:"id" json:"id"`
	Username       string    `db:"username" json:"username"`
	TotalTraffic   int64     `db:"total_traffic" json:"total_traffic"`
	TodayTraffic   int64     `db:"today_traffic" json:"today_traffic"`
	HistoryTraffic string    `db:"history_traffic" json:"history_traffic"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// ProxyTrafficInfo 代理流量信息
type ProxyTrafficInfo struct {
	Name            string `json:"name"`
	TodayTrafficIn  int64  `json:"todayTrafficIn"`
	TodayTrafficOut int64  `json:"todayTrafficOut"`
}

// ProxyResponse API响应
type ProxyResponse struct {
	Proxies []ProxyTrafficInfo `json:"proxies"`
}
