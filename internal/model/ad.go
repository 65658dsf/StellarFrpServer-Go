package model

import "time"

// Ad 广告模型
type Ad struct {
	ID          int64     `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	Image       string    `db:"image" json:"image"`
	LinkURL     string    `db:"link_url" json:"linkUrl"`
	StartTime   time.Time `db:"start_time" json:"startTime"`
	EndTime     time.Time `db:"end_time" json:"endTime"`
	IsActive    bool      `db:"is_active" json:"isActive"`
	Priority    int       `db:"priority" json:"priority"`
	CreatedAt   time.Time `db:"created_at" json:"-"`
	UpdatedAt   time.Time `db:"updated_at" json:"-"`
}
