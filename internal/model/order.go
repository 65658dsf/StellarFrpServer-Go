package model

import (
	"database/sql"
	"time"
)

// Order 订单模型
type Order struct {
	ID             uint64         `db:"id" json:"id"`
	OrderNo        string         `db:"order_no" json:"order_no"`
	UserID         uint64         `db:"user_id" json:"user_id"`
	ProductID      uint64         `db:"product_id" json:"product_id"`
	ProductSkuID   string         `db:"product_sku_id" json:"product_sku_id"`
	ProductName    string         `db:"product_name" json:"product_name"`
	Amount         float64        `db:"amount" json:"amount"`
	Status         int            `db:"status" json:"status"` // 0: 待支付, 1: 已支付, 2: 已取消, 3: 已退款
	Remark         string         `db:"remark" json:"remark"`
	AfdianTradeNo  sql.NullString `db:"afdian_trade_no" json:"afdian_trade_no,omitempty"`
	RewardAction   sql.NullString `db:"reward_action" json:"reward_action,omitempty"`
	RewardValue    sql.NullString `db:"reward_value" json:"reward_value,omitempty"`
	RewardExecuted bool           `db:"reward_executed" json:"reward_executed"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
	PaidAt         sql.NullTime   `db:"paid_at" json:"paid_at,omitempty"`
}
