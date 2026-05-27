// Package repository 提供行程历史表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// UpsertTrip 创建或更新行程记录。
func (s *Store) UpsertTrip(ctx context.Context, trip model.Trip) (*model.Trip, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	ts := now()
	if trip.CreatedAt.IsZero() {
		trip.CreatedAt = ts
	}
	trip.UpdatedAt = ts
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "trip_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "session_id", "origin", "destination", "start_date", "end_date",
			"purpose", "itinerary_json", "raw_json", "updated_at",
		}),
	}).Create(&trip).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("trip_id = ?", trip.TripID).First(&trip).Error
	return &trip, err
}

// GetTrip 查询用户指定行程。
func (s *Store) GetTrip(ctx context.Context, userID, tripID string) (*model.Trip, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	var trip model.Trip
	err := s.db.WithContext(ctx).Where("user_id = ? AND trip_id = ?", userID, tripID).First(&trip).Error
	return &trip, err
}

// ListTrips 按用户分页查询历史行程。
func (s *Store) ListTrips(ctx context.Context, userID string, limit, offset int) ([]model.Trip, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var trips []model.Trip
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&trips).Error
	return trips, err
}

// DeleteTrip 删除用户指定行程。
func (s *Store) DeleteTrip(ctx context.Context, userID, tripID string) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	result := s.db.WithContext(ctx).Where("user_id = ? AND trip_id = ?", userID, tripID).Delete(&model.Trip{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
