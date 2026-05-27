// Package repository 提供用户统计表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// UpsertStatistics 创建或更新用户统计信息。
func (s *Store) UpsertStatistics(ctx context.Context, stats model.UserStatistics) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	ts := now()
	if stats.CreatedAt.IsZero() {
		stats.CreatedAt = ts
	}
	stats.UpdatedAt = ts
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_trips", "total_messages", "frequent_destinations_json", "updated_at",
		}),
	}).Create(&stats).Error
}
