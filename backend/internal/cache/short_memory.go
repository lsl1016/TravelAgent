// Package cache 提供会话短期记忆的 Redis List 实现。
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	shortMemoryTTL = time.Hour
	maxMessages    = int64(20)
)

// RecentMessage 是写入 Redis 短期记忆的轻量消息结构。
type RecentMessage struct {
	MessageID string         `json:"message_id,omitempty"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// GetRecentMessages 从 Redis 读取指定会话的最近消息。
func (c *Cache) GetRecentMessages(ctx context.Context, sessionID string) ([]RecentMessage, error) {
	if c == nil || c.client == nil {
		return nil, ErrUnavailable
	}
	items, err := c.client.LRange(ctx, sessionMessagesKey(sessionID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	messages := make([]RecentMessage, 0, len(items))
	for _, item := range items {
		var msg RecentMessage
		if err := json.Unmarshal([]byte(item), &msg); err == nil {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// AppendRecentMessages 追加会话消息，并保持最多 20 条的滑动窗口。
func (c *Cache) AppendRecentMessages(ctx context.Context, sessionID string, messages ...RecentMessage) error {
	if c == nil || c.client == nil {
		return ErrUnavailable
	}
	if len(messages) == 0 {
		return nil
	}
	key := sessionMessagesKey(sessionID)
	pipe := c.client.TxPipeline()
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		pipe.RPush(ctx, key, data)
	}
	pipe.LTrim(ctx, key, -maxMessages, -1)
	pipe.Expire(ctx, key, shortMemoryTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// ReplaceRecentMessages 用 MySQL 回源结果重建 Redis 短期记忆。
func (c *Cache) ReplaceRecentMessages(ctx context.Context, sessionID string, messages []RecentMessage) error {
	if c == nil || c.client == nil {
		return ErrUnavailable
	}
	key := sessionMessagesKey(sessionID)
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, key)
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		pipe.RPush(ctx, key, data)
	}
	pipe.LTrim(ctx, key, -maxMessages, -1)
	pipe.Expire(ctx, key, shortMemoryTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// sessionMessagesKey 生成短期记忆 Redis key。
func sessionMessagesKey(sessionID string) string {
	return fmt.Sprintf("session:%s:messages", sessionID)
}
