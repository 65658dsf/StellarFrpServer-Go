package repository

import (
	"context"
	"stellarfrp/internal/model"

	"github.com/jmoiron/sqlx"
)

// AnnouncementRepository 公告存储库
type AnnouncementRepository struct {
	db *sqlx.DB
}

// NewAnnouncementRepository 创建公告存储库实例
func NewAnnouncementRepository(db *sqlx.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

// GetAnnouncements 获取公告列表（分页）
func (r *AnnouncementRepository) GetAnnouncements(ctx context.Context, page, limit int) ([]model.Announcement, error) {
	var announcements []model.Announcement
	offset := (page - 1) * limit

	query := `
		SELECT * FROM announcements 
		WHERE is_visible = true
		ORDER BY is_important DESC, publish_date DESC
		LIMIT ? OFFSET ?
	`
	err := r.db.SelectContext(ctx, &announcements, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return announcements, nil
}

// GetAnnouncementByID 根据ID获取公告
func (r *AnnouncementRepository) GetAnnouncementByID(ctx context.Context, id int64) (*model.Announcement, error) {
	var announcement model.Announcement
	query := "SELECT * FROM announcements WHERE id = ?"
	err := r.db.GetContext(ctx, &announcement, query, id)
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

// CountAnnouncements 获取公告总数
func (r *AnnouncementRepository) CountAnnouncements(ctx context.Context) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM announcements WHERE is_visible = true"
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CreateAnnouncement 创建公告
func (r *AnnouncementRepository) CreateAnnouncement(ctx context.Context, a *model.Announcement) error {
	query := `INSERT INTO announcements (title, content, is_important, is_visible, publish_date, author) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, a.Title, a.Content, a.IsImportant, a.IsVisible, a.PublishDate, a.Author)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err == nil {
		a.ID = id
	}
	return err
}

// UpdateAnnouncement 更新公告
func (r *AnnouncementRepository) UpdateAnnouncement(ctx context.Context, a *model.Announcement) error {
	query := `UPDATE announcements SET title=?, content=?, is_important=?, is_visible=?, author=?, publish_date=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, a.Title, a.Content, a.IsImportant, a.IsVisible, a.Author, a.PublishDate, a.ID)
	return err
}

// DeleteAnnouncement 删除公告
func (r *AnnouncementRepository) DeleteAnnouncement(ctx context.Context, id int64) error {
	query := `DELETE FROM announcements WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetAnnouncementsAdmin 获取所有公告（含不可见）
func (r *AnnouncementRepository) GetAnnouncementsAdmin(ctx context.Context, page, limit int) ([]model.Announcement, error) {
	var announcements []model.Announcement
	offset := (page - 1) * limit
	query := `SELECT * FROM announcements ORDER BY is_important DESC, publish_date DESC LIMIT ? OFFSET ?`
	err := r.db.SelectContext(ctx, &announcements, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return announcements, nil
}

// CountAnnouncementsAdmin 获取所有公告总数（含不可见）
func (r *AnnouncementRepository) CountAnnouncementsAdmin(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM announcements`
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}
