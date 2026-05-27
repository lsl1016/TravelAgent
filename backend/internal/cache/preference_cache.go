// 提供偏好、摘要和 Agent 运行状态缓存。
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	preferenceTTL = 30 * time.Minute
	summaryTTL    = 6 * time.Hour
	agentRunTTL   = 24 * time.Hour
)

func (c *Cache) GetPreferences(ctx context.Context, userID string) (map[string]json.RawMessage, error) {
	data, err := c.getBytes(ctx, preferencesKey(userID))
	if err != nil || data == nil {
		return nil, err
	}
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}

// 缓存用户偏好热点数据。
func (c *Cache) SetPreferences(ctx context.Context, userID string, prefs map[string]json.RawMessage) error {
	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	return c.setJSON(ctx, preferencesKey(userID), data, preferenceTTL)
}

// 删除用户偏好缓存，下一次读取回源 MySQL。
func (c *Cache) InvalidatePreferences(ctx context.Context, userID string) error {
	return c.delete(ctx, preferencesKey(userID))
}

// 读取用户长期记忆摘要缓存。
func (c *Cache) GetSummary(ctx context.Context, userID string) (string, error) {
	data, err := c.getBytes(ctx, summaryKey(userID))
	if err != nil || data == nil {
		return "", err
	}
	return string(data), nil
}

// 写入用户长期记忆摘要缓存。
func (c *Cache) SetSummary(ctx context.Context, userID, summary string) error {
	return c.setJSON(ctx, summaryKey(userID), []byte(summary), summaryTTL)
}

// 删除用户摘要缓存。
func (c *Cache) InvalidateSummary(ctx context.Context, userID string) error {
	return c.delete(ctx, summaryKey(userID))
}

// 缓存 Agent 运行状态，便于排障和异步状态查询。
func (c *Cache) SetAgentRunStatus(ctx context.Context, runID string, status map[string]any) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return c.setJSON(ctx, agentRunStatusKey(runID), data, agentRunTTL)
}

// 生成用户偏好缓存 key。
func preferencesKey(userID string) string {
	return fmt.Sprintf("user:%s:preferences", userID)
}

// 生成用户摘要缓存 key。
func summaryKey(userID string) string {
	return fmt.Sprintf("user:%s:summary", userID)
}

// 生成 Agent 运行状态缓存 key。
func agentRunStatusKey(runID string) string {
	return fmt.Sprintf("agent_run:%s:status", runID)
}
