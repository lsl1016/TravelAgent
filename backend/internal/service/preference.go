// Package service 实现用户偏好读写逻辑。
package service

import (
	"context"
	"encoding/json"
	"net/http"

	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/repository"
)

// PreferenceService 负责用户偏好的读取、更新和缓存失效。
type PreferenceService struct {
	store *repository.Store
	cache *cache.Cache
}

// NewPreferenceService 创建偏好服务。
func NewPreferenceService(store *repository.Store, cache *cache.Cache) *PreferenceService {
	return &PreferenceService{store: store, cache: cache}
}

// List 查询用户偏好，优先读取 Redis，未命中时回源 MySQL。
func (s *PreferenceService) List(ctx context.Context, userID string) ([]model.PreferenceDTO, error) {
	if userID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	if cached, err := s.cache.GetPreferences(ctx, userID); err == nil && cached != nil {
		dtos := make([]model.PreferenceDTO, 0, len(cached))
		for typ, value := range cached {
			dtos = append(dtos, model.PreferenceDTO{Type: typ, Value: value})
		}
		return dtos, nil
	}
	prefs, err := s.store.GetPreferences(ctx, userID)
	if err != nil {
		return nil, MapError(err)
	}
	_ = s.cache.SetPreferences(ctx, userID, preferenceMap(prefs))
	return preferenceDTOs(prefs), nil
}

// Put 批量更新用户偏好，支持 replace 和 append。
func (s *PreferenceService) Put(ctx context.Context, userID string, items []model.PreferenceDTO) ([]model.PreferenceDTO, error) {
	if userID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	if len(items) == 0 {
		return nil, InvalidArgument("preferences is required")
	}
	for _, item := range items {
		if _, err := s.apply(ctx, userID, item.Type, item.Value, item.Action, item.Source); err != nil {
			return nil, err
		}
	}
	_ = s.cache.InvalidatePreferences(ctx, userID)
	return s.List(ctx, userID)
}

// Patch 更新单个偏好类型。
func (s *PreferenceService) Patch(ctx context.Context, userID, prefType string, raw json.RawMessage, action, source string) (*model.PreferenceDTO, error) {
	if userID == "" || prefType == "" {
		return nil, InvalidArgument("user_id and preference type are required")
	}
	pref, err := s.apply(ctx, userID, prefType, raw, action, source)
	if err != nil {
		return nil, err
	}
	_ = s.cache.InvalidatePreferences(ctx, userID)
	dto := model.PreferenceDTO{Type: pref.Type, Value: json.RawMessage(pref.ValueJSON), Source: pref.Source}
	return &dto, nil
}

// Delete 删除单个偏好类型并失效缓存。
func (s *PreferenceService) Delete(ctx context.Context, userID, prefType string) error {
	if userID == "" || prefType == "" {
		return InvalidArgument("user_id and preference type are required")
	}
	if err := s.store.DeletePreference(ctx, userID, prefType); err != nil {
		return MapError(err)
	}
	_ = s.cache.InvalidatePreferences(ctx, userID)
	return nil
}

// apply 执行单条偏好的校验、追加合并和 upsert。
func (s *PreferenceService) apply(ctx context.Context, userID, prefType string, raw json.RawMessage, action, source string) (*model.Preference, error) {
	if prefType == "" {
		return nil, InvalidArgument("preference type is required")
	}
	if err := validateRawJSON(raw); err != nil {
		return nil, NewError(http.StatusBadRequest, CodeInvalidArgument, "preference value must be valid json", err)
	}
	value := raw
	if normalizeAction(action) == "append" {
		current, err := s.store.GetPreference(ctx, userID, prefType)
		if err == nil {
			merged, err := appendPreferenceValue(json.RawMessage(current.ValueJSON), raw)
			if err != nil {
				return nil, NewError(http.StatusBadRequest, CodeInvalidArgument, "preference append failed", err)
			}
			value = merged
		}
	}
	pref, err := s.store.UpsertPreference(ctx, userID, prefType, rawToJSON(value), source)
	if err != nil {
		return nil, MapError(err)
	}
	return pref, nil
}
