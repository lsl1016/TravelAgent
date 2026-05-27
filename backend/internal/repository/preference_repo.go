// Package repository 提供用户偏好表访问方法。
package repository

import (
	"context"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// GetPreferences 查询用户全部偏好。
func (s *Store) GetPreferences(ctx context.Context, userID string) ([]model.Preference, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	var prefs []model.Preference
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("type ASC").Find(&prefs).Error
	return prefs, err
}

// GetPreference 查询用户某一种偏好。
func (s *Store) GetPreference(ctx context.Context, userID, prefType string) (*model.Preference, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	var pref model.Preference
	err := s.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, prefType).First(&pref).Error
	return &pref, err
}

// UpsertPreference 创建或覆盖用户偏好。
func (s *Store) UpsertPreference(ctx context.Context, userID, prefType string, value datatypes.JSON, source string) (*model.Preference, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if source == "" {
		source = "agent"
	}
	ts := now()
	pref := model.Preference{
		UserID:    userID,
		Type:      prefType,
		ValueJSON: value,
		Source:    source,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "type"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value_json": value,
			"source":     source,
			"updated_at": ts,
		}),
	}).Create(&pref).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, prefType).First(&pref).Error
	return &pref, err
}

// DeletePreference 删除用户某一种偏好。
func (s *Store) DeletePreference(ctx context.Context, userID, prefType string) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	result := s.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, prefType).Delete(&model.Preference{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
