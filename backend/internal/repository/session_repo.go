// Package repository 提供会话表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// EnsureSession 确保会话存在，并在重复请求时刷新活跃状态。
func (s *Store) EnsureSession(ctx context.Context, sessionID, userID, title string) (*model.Session, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	ts := now()
	session := model.Session{
		SessionID: sessionID,
		UserID:    userID,
		Title:     title,
		Status:    "active",
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"updated_at": ts,
			"status":     "active",
		}),
	}).Create(&session).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	return &session, err
}

// CreateSession 创建新的会话记录。
func (s *Store) CreateSession(ctx context.Context, session model.Session) (*model.Session, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	ts := now()
	session.CreatedAt = ts
	session.UpdatedAt = ts
	if session.Status == "" {
		session.Status = "active"
	}
	err := s.db.WithContext(ctx).Create(&session).Error
	return &session, err
}

// ListSessions 按用户分页查询会话。
func (s *Store) ListSessions(ctx context.Context, userID string, limit, offset int) ([]model.Session, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var sessions []model.Session
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error
	return sessions, err
}

// DeleteSession 将会话标记为删除并写入结束时间。
func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	ts := now()
	result := s.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{"status": "deleted", "ended_at": ts, "updated_at": ts})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
