// Package repository 提供聊天消息表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// CreateMessage 写入聊天消息，message_id 冲突时保持幂等。
func (s *Store) CreateMessage(ctx context.Context, msg model.ChatMessage) (*model.ChatMessage, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now()
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}},
		DoNothing: true,
	}).Create(&msg).Error
	return &msg, err
}

// ListMessages 按会话分页查询消息，支持 before 游标。
func (s *Store) ListMessages(ctx context.Context, sessionID string, limit int, beforeMessageID string) ([]model.ChatMessage, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := s.db.WithContext(ctx).Where("session_id = ?", sessionID)
	if beforeMessageID != "" {
		var before model.ChatMessage
		err := s.db.WithContext(ctx).Where("message_id = ?", beforeMessageID).First(&before).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return []model.ChatMessage{}, nil
			}
			return nil, err
		}
		query = query.Where("created_at < ?", before.CreatedAt)
	}
	var messages []model.ChatMessage
	err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error
	return reverseMessages(messages), err
}

// reverseMessages 将倒序查询结果恢复成时间正序。
func reverseMessages(messages []model.ChatMessage) []model.ChatMessage {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages
}
