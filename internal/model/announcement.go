package model

import "time"

// Announcement 公告模型
type Announcement struct {
	ID          int64     `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Content     string    `db:"content" json:"content"`
	IsImportant bool      `db:"is_important" json:"is_important"`
	IsVisible   bool      `db:"is_visible" json:"is_visible"`
	PublishDate time.Time `db:"publish_date" json:"publish_date"`
	Author      string    `db:"author" json:"author"`
	CreatedAt   time.Time `db:"created_at" json:"-"`
	UpdatedAt   time.Time `db:"updated_at" json:"-"`
}

// PaginatedAnnouncements 分页公告结果
type PaginatedAnnouncements struct {
	Total int64          `json:"total"`
	Items []Announcement `json:"items"`
}
