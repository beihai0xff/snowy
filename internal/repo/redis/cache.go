package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Cache 通用缓存接口，供上层业务依赖。
type Cache interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// CacheStore 通用缓存存储，参考技术方案 §18.3.1。
// 实现 Cache 接口。
type CacheStore struct {
	client *goredis.Client
}

// NewCacheStore 创建缓存存储。
func NewCacheStore(client *goredis.Client) *CacheStore {
	return &CacheStore{client: client}
}

// Get 获取缓存值并反序列化。
func (c *CacheStore) Get(ctx context.Context, key string, dest any) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("cache get %s: %w", key, err)
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set 序列化并存入缓存。
func (c *CacheStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete 删除缓存。
func (c *CacheStore) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists 检查缓存是否存在。
func (c *CacheStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}
