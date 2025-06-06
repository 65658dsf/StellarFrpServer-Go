package repository

import (
	"stellarfrp/internal/model"
	"time"

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

// CreateProduct 创建商品
func (r *ProductRepository) CreateProduct(product *model.Product) error {
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	query := `
		INSERT INTO products (
			sku_id, name, description, price, plan_id, is_active, 
			created_at, updated_at, reward_action, reward_value
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(
		query,
		product.SkuID,
		product.Name,
		product.Description,
		product.Price,
		product.PlanID,
		product.IsActive,
		product.CreatedAt,
		product.UpdatedAt,
		product.RewardAction,
		product.RewardValue,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	product.ID = uint64(id)
	return nil
}

// UpdateProduct 更新商品
func (r *ProductRepository) UpdateProduct(product *model.Product) error {
	product.UpdatedAt = time.Now()

	query := `
		UPDATE products 
		SET sku_id = ?, name = ?, description = ?, price = ?, 
			plan_id = ?, is_active = ?, updated_at = ?, 
			reward_action = ?, reward_value = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(
		query,
		product.SkuID,
		product.Name,
		product.Description,
		product.Price,
		product.PlanID,
		product.IsActive,
		product.UpdatedAt,
		product.RewardAction,
		product.RewardValue,
		product.ID,
	)
	return err
}

// DeleteProduct 删除商品（软删除，将is_active设为0）
func (r *ProductRepository) DeleteProduct(id uint64) error {
	query := `UPDATE products SET is_active = 0, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), id)
	return err
}

// GetProductsWithPagination 获取分页商品列表
func (r *ProductRepository) GetProductsWithPagination(page, pageSize int) ([]model.Product, int, error) {
	// 先获取总记录数
	countQuery := `SELECT COUNT(*) FROM products WHERE is_active = 1`
	var total int
	err := r.db.Get(&total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有记录，直接返回空数组和0
	if total == 0 {
		return []model.Product{}, 0, nil
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	var products []model.Product
	query := `SELECT * FROM products WHERE is_active = 1 ORDER BY id DESC LIMIT ? OFFSET ?`
	err = r.db.Select(&products, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
