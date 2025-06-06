package repository

import (
	"stellarfrp/internal/model"

	"github.com/jmoiron/sqlx"
)

// ProductRepository 商品存储库
type ProductRepository struct {
	db *sqlx.DB
}

// NewProductRepository 创建商品存储库
func NewProductRepository(db *sqlx.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetProducts 获取所有商品
func (r *ProductRepository) GetProducts() ([]model.Product, error) {
	var products []model.Product
	query := `SELECT * FROM products WHERE is_active = 1`
	err := r.db.Select(&products, query)
	return products, err
}

// GetProductByID 根据ID获取商品
func (r *ProductRepository) GetProductByID(id uint64) (*model.Product, error) {
	var product model.Product
	query := `SELECT * FROM products WHERE id = ? AND is_active = 1`
	err := r.db.Get(&product, query, id)
	return &product, err
}

// GetProductBySkuID 根据SkuID获取商品
func (r *ProductRepository) GetProductBySkuID(skuID string) (*model.Product, error) {
	var product model.Product
	query := `SELECT * FROM products WHERE sku_id = ? AND is_active = 1`
	err := r.db.Get(&product, query, skuID)
	return &product, err
}
