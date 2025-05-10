package model

// SystemStatus 系统状态
type SystemStatus struct {
	TotalUsers   int64  `json:"total_users"`
	TotalTunnels int64  `json:"total_tunnels"`
	TotalTraffic string `json:"total_traffic"`
	TotalNodes   int64  `json:"total_nodes"`
}
