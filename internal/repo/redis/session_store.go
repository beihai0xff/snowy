package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// SessionContext 会话上下文接口，供上层业务依赖。
type SessionContext interface {
	Get(ctx context.Context, sessionID string) (map[string]any, error)
	Set(ctx context.Context, sessionID string, data map[string]any) error
	Delete(ctx context.Context, sessionID string) error
}

// SessionStore 会话上下文存储，支持滑动窗口 TTL。
// Key: session:{session_id}:ctx，TTL: 30min 滑动。
// 参考技术方案 §18.3.1。
// 实现 SessionContext 接口。
type SessionStore struct {
	client     *goredis.Client
	defaultTTL time.Duration
}

// NewSessionStore 创建会话存储。
func NewSessionStore(client *goredis.Client) *SessionStore {
	return &SessionStore{
		client:     client,
		defaultTTL: 30 * time.Minute,
	}
}

// Get 获取会话上下文，并续期 TTL。
func (s *SessionStore) Get(ctx context.Context, sessionID string) (map[string]any, error) {
	k := s.key(sessionID)

	val, err := s.client.Get(ctx, k).Result()
	if err != nil {
		return nil, fmt.Errorf("session get: %w", err)
	}

	// 滑动窗口续期
	s.client.Expire(ctx, k, s.defaultTTL)

	var data map[string]any
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("session unmarshal: %w", err)
	}

	return data, nil
}

// Set 保存会话上下文。
func (s *SessionStore) Set(ctx context.Context, sessionID string, data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}

	return s.client.Set(ctx, s.key(sessionID), b, s.defaultTTL).Err()
}

// Delete 删除会话上下文。
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, s.key(sessionID)).Err()
}

func (s *SessionStore) key(sessionID string) string {
	return fmt.Sprintf("session:%s:ctx", sessionID)
}
