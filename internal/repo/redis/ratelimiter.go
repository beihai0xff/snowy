package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RateLimiter 基于 Redis 滑动窗口的限流器。
// 参考技术方案 §18A.4。
type RateLimiter struct {
	client *goredis.Client
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(client *goredis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow 检查是否允许请求。
// key: rate:{user_id}:{window} 格式
// limit: 窗口内最大请求数
// window: 滑动窗口大小
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()
	member := fmt.Sprintf("%d-%d", now, time.Now().UnixNano())

	pipe := r.client.Pipeline()

	// 移除窗口外的旧记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	// 添加当前请求
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now), Member: member})
	// 计算窗口内请求数
	countCmd := pipe.ZCard(ctx, key)
	// 设置 key 过期
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit pipeline: %w", err)
	}

	count := countCmd.Val()
	return count <= int64(limit), nil
}
