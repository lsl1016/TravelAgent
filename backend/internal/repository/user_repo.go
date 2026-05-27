// Package repository 提供用户表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// EnsureUser 确保用户存在，不存在时创建本地来源用户。
func (s *Store) EnsureUser(ctx context.Context, userID string) (*model.User, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	ts := now()
	user := model.User{
		UserID:    userID,
		Source:    "local",
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"updated_at": ts}),
	}).Create(&user).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	return &user, err
}
