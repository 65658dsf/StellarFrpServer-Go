package repository

import (
	"database/sql"
	"fmt"
	"stellarfrp/internal/model"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// OrderRepository 订单存储库
type OrderRepository struct {
	db *sqlx.DB
}

// NewOrderRepository 创建订单存储库
func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder 创建订单
func (r *OrderRepository) CreateOrder(order *model.Order) error {
	query := `
		INSERT INTO orders (
			order_no, user_id, product_id, product_sku_id, product_name, 
			amount, status, remark, reward_action, reward_value, reward_executed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(
		query,
		order.OrderNo,
		order.UserID,
		order.ProductID,
		order.ProductSkuID,
		order.ProductName,
		order.Amount,
		order.Status,
		order.Remark,
		order.RewardAction,
		order.RewardValue,
		order.RewardExecuted,
	)
	return err
}

// GetOrderByOrderNo 根据订单号获取订单
func (r *OrderRepository) GetOrderByOrderNo(orderNo string) (*model.Order, error) {
	var order model.Order
	query := `SELECT * FROM orders WHERE order_no = ?`
	err := r.db.Get(&order, query, orderNo)
	return &order, err
}

// GetOrderByAfdianTradeNo 根据爱发电订单号获取订单
func (r *OrderRepository) GetOrderByAfdianTradeNo(afdianTradeNo string) (*model.Order, error) {
	var order model.Order
	query := `SELECT * FROM orders WHERE afdian_trade_no = ?`
	err := r.db.Get(&order, query, afdianTradeNo)
	return &order, err
}

// UpdateOrderStatus 更新订单状态
func (r *OrderRepository) UpdateOrderStatus(orderNo string, status int, afdianTradeNo string) error {
	query := `
		UPDATE orders 
		SET status = ?, afdian_trade_no = ?, paid_at = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE order_no = ?
	`
	// 创建一个可空的字符串
	nullTradeNo := sql.NullString{
		String: afdianTradeNo,
		Valid:  afdianTradeNo != "",
	}

	// 创建一个可空的时间
	var paidAt sql.NullTime
	if status == 1 { // 已支付
		paidAt = sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		}
	}

	_, err := r.db.Exec(query, status, nullTradeNo, paidAt, orderNo)
	return err
}

// UpdateRewardExecuted 更新奖励执行状态
func (r *OrderRepository) UpdateRewardExecuted(orderNo string, executed bool) error {
	query := `
		UPDATE orders 
		SET reward_executed = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE order_no = ?
	`
	_, err := r.db.Exec(query, executed, orderNo)
	return err
}

// GetUnexecutedRewards 获取未执行奖励的已支付订单
func (r *OrderRepository) GetUnexecutedRewards() ([]model.Order, error) {
	var orders []model.Order
	query := `
		SELECT * FROM orders 
		WHERE status = 1 AND reward_executed = 0
	`
	err := r.db.Select(&orders, query)
	return orders, err
}

// GetOrdersByUserID 获取用户的所有订单
func (r *OrderRepository) GetOrdersByUserID(userID uint64, page, pageSize int) ([]model.Order, int, error) {
	// 先获取总记录数
	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = ?`
	var total int
	err := r.db.Get(&total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有记录，直接返回空数组和0
	if total == 0 {
		return []model.Order{}, 0, nil
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	var orders []model.Order
	query := `SELECT * FROM orders WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err = r.db.Select(&orders, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetOrderByRemarkAndStatus 根据备注和状态获取订单
func (r *OrderRepository) GetOrderByRemarkAndStatus(remark string, status int) (*model.Order, error) {
	var order model.Order
	query := `SELECT * FROM orders WHERE remark = ? AND status = ?`
	err := r.db.Get(&order, query, remark, status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// GetOrdersWithFilter 获取带过滤条件的订单列表
func (r *OrderRepository) GetOrdersWithFilter(page, pageSize int, userID *uint64, status *int, orderNo string) ([]model.Order, int, error) {
	// 构建查询条件
	conditions := []string{}
	args := []interface{}{}

	if userID != nil {
		conditions = append(conditions, "user_id = ?")
		args = append(args, *userID)
	}

	if status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *status)
	}

	if orderNo != "" {
		conditions = append(conditions, "order_no LIKE ?")
		args = append(args, "%"+orderNo+"%")
	}

	// 构建WHERE子句
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 获取总记录数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereClause)
	var total int
	err := r.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有记录，直接返回空数组和0
	if total == 0 {
		return []model.Order{}, 0, nil
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	query := fmt.Sprintf("SELECT * FROM orders %s ORDER BY created_at DESC LIMIT ? OFFSET ?", whereClause)
	var orders []model.Order

	// 添加分页参数
	queryArgs := append(args, pageSize, offset)
	err = r.db.Select(&orders, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateOrderPaidTime 更新订单支付时间
func (r *OrderRepository) UpdateOrderPaidTime(orderNo string, paidTime time.Time) error {
	query := `
		UPDATE orders 
		SET paid_at = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE order_no = ?
	`
	paidAtSQL := sql.NullTime{
		Time:  paidTime,
		Valid: true,
	}
	_, err := r.db.Exec(query, paidAtSQL, orderNo)
	return err
}

// DeleteOrder 删除订单
func (r *OrderRepository) DeleteOrder(orderNo string) error {
	query := `DELETE FROM orders WHERE order_no = ?`
	_, err := r.db.Exec(query, orderNo)
	return err
}
