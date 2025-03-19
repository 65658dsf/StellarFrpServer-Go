package database

import (
	"context"
	"fmt"

	"stellarfrp/config"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient 创建一个新的Redis客户端
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       0, // 使用默认数据库
	})

	// 验证连接
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
