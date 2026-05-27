// Package cache 封装 Redis 连接和缓存操作。
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrUnavailable = errors.New("redis unavailable")

// Cache 包装 Redis 客户端，并在未配置 Redis 时提供可识别的降级状态。
type Cache struct {
	client *redis.Client
}

// Open 根据配置创建 Redis 客户端，未配置地址时返回降级空缓存。
func Open(addr, password string, db int) *Cache {
	if addr == "" {
		return &Cache{}
	}
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

// 返回底层 Redis 客户端，供少量高级场景复用。
func (c *Cache) Client() *redis.Client {
	if c == nil {
		return nil
	}
	return c.client
}

// 检查 Redis 是否可访问。
func (c *Cache) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return ErrUnavailable
	}
	return c.client.Ping(ctx).Err()
}

// 写入带 TTL 的 JSON 缓存。
func (c *Cache) setJSON(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return ErrUnavailable
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// 读取缓存字节，缓存未命中时返回 nil。
func (c *Cache) getBytes(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, ErrUnavailable
	}
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return data, err
}

// 删除一个或多个缓存 key。
func (c *Cache) delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil {
		return ErrUnavailable
	}
	return c.client.Del(ctx, keys...).Err()
}
