package model

import (
	"time"
)

// Product 商品模型
type Product struct {
	ID           uint64    `db:"id" json:"id"`
	SkuID        string    `db:"sku_id" json:"sku_id"`
	Name         string    `db:"name" json:"name"`
	Description  string    `db:"description" json:"description"`
	Price        float64   `db:"price" json:"price"`
	PlanID       string    `db:"plan_id" json:"plan_id"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	RewardAction string    `db:"reward_action" json:"reward_action"`
	RewardValue  string    `db:"reward_value" json:"reward_value"`
}
