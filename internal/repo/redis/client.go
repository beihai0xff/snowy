// Package redis 提供 Redis 连接及缓存、限流、会话存储（基础设施层）。
package redis

import (
	"context"
	"fmt"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	goredis "github.com/redis/go-redis/v9"
)

// NewClient 创建 Redis 客户端。
func NewClient(cfg config.RedisConfig) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
