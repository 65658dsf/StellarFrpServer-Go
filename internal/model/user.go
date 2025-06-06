package model

import "time"

// User 用户模型
type User struct {
	ID                uint64     `db:"id" json:"id"`
	Username          string     `db:"username" json:"username"`
	Email             string     `db:"email" json:"email"`
	Password          string     `db:"password" json:"-"`
	RegisterTime      time.Time  `db:"register_time" json:"register_time"`
	GroupID           int64      `db:"group_id" json:"group_id"`
	IsVerified        int        `db:"is_verified" json:"is_verified"`
	VerifyCount       int        `db:"verify_count" json:"verify_count"`
	Status            int        `db:"status" json:"status"`
	Token             string     `db:"token" json:"-"`
	GroupTime         *time.Time `db:"group_time" json:"group_time"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
	TunnelCount       *int       `db:"tunnel_count" json:"tunnel_count"`
	Bandwidth         *int       `db:"bandwidth" json:"bandwidth"`
	TrafficQuota      *int64     `db:"traffic_quota" json:"traffic_quota"`
	LastCheckin       *time.Time `db:"last_checkin" json:"last_checkin"`
	CheckinCount      int        `db:"checkin_count" json:"checkin_count"`
	ContinuityCheckin int        `db:"continuity_checkin" json:"continuity_checkin"`
}
