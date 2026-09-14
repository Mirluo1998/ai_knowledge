// Package cache 提供基于 Redis 的缓存实现。
// 实现 service 层定义的 SessionStore 接口（accept interfaces, return structs）。
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// sessionKeyPrefix 是登录会话在 Redis 中的 key 前缀。
const sessionKeyPrefix = "login:token:"

// RedisSessionStore 是会话存储的 Redis 实现。
type RedisSessionStore struct {
	client *redis.Client
}

// NewRedisSessionStore 创建会话存储实例。
func NewRedisSessionStore(client *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

// Set 以 token 为 key 写入用户信息（JSON），并设置过期时间。
func (s *RedisSessionStore) Set(ctx context.Context, token string, user *model.UserResponse, ttl time.Duration) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal session user: %w", err)
	}
	if err := s.client.Set(ctx, sessionKeyPrefix+token, data, ttl).Err(); err != nil {
		return fmt.Errorf("set session: %w", err)
	}
	return nil
}

// Get 按 token 读取用户信息；会话不存在或已过期时返回 service.ErrSessionNotFound。
func (s *RedisSessionStore) Get(ctx context.Context, token string) (*model.UserResponse, error) {
	data, err := s.client.Get(ctx, sessionKeyPrefix+token).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var user model.UserResponse
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal session user: %w", err)
	}
	return &user, nil
}
