package repository

import (
	"context"
	"stellarfrp/internal/model"

	"github.com/jmoiron/sqlx"
)

// AdRepository 广告存储库
type AdRepository struct {
	db *sqlx.DB
}

// NewAdRepository 创建广告存储库实例
func NewAdRepository(db *sqlx.DB) *AdRepository {
	return &AdRepository{db: db}
}

// GetAds 获取所有广告
func (r *AdRepository) GetAds(ctx context.Context) ([]model.Ad, error) {
	var ads []model.Ad
	query := `
		SELECT * FROM ads 
		WHERE is_active = true
		ORDER BY id DESC
	`
	err := r.db.SelectContext(ctx, &ads, query)
	if err != nil {
		return nil, err
	}
	return ads, nil
}

// GetActiveAds 获取所有活跃的广告
func (r *AdRepository) GetActiveAds(ctx context.Context) ([]*model.Ad, error) {
	var ads []*model.Ad
	query := `
		SELECT * FROM ads 
		WHERE is_active = true
		ORDER BY id DESC
	`
	err := r.db.SelectContext(ctx, &ads, query)
	if err != nil {
		return nil, err
	}
	return ads, nil
}
