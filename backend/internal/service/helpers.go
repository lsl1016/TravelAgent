// Package service 提供服务层复用的转换和校验工具。
package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/model"
)

// 生成带业务前缀的 UUID。
func newID(prefix string) string {
	return prefix + "_" + uuid.NewString()
}

// 将请求中的 RawMessage 转成 GORM JSON 类型。
func rawToJSON(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(raw)
}

// 将任意值序列化为 JSON，失败时返回 null。
func mustJSON(value any) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(data)
}

// 解析通用分页参数。
func parseLimitOffset(limitText, offsetText string, defaultLimit int) (int, int) {
	limit := defaultLimit
	offset := 0
	if limitText != "" {
		if parsed, err := strconv.Atoi(limitText); err == nil {
			limit = parsed
		}
	}
	if offsetText != "" {
		if parsed, err := strconv.Atoi(offsetText); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

// 解析 YYYY-MM-DD 日期。
func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// 将可空日期格式化成 API 字符串。
func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

// 将偏好实体转换成 Agent 上下文需要的 map。
func preferenceMap(prefs []model.Preference) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(prefs))
	for _, pref := range prefs {
		result[pref.Type] = json.RawMessage(pref.ValueJSON)
	}
	return result
}

// 将偏好实体转换成 API DTO。
func preferenceDTOs(prefs []model.Preference) []model.PreferenceDTO {
	result := make([]model.PreferenceDTO, 0, len(prefs))
	for _, pref := range prefs {
		result = append(result, model.PreferenceDTO{
			Type:   pref.Type,
			Value:  json.RawMessage(pref.ValueJSON),
			Source: pref.Source,
		})
	}
	return result
}

// 将消息实体转换成 API DTO。
func messageDTO(msg model.ChatMessage) model.MessageDTO {
	return model.MessageDTO{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		UserID:    msg.UserID,
		Role:      msg.Role,
		Content:   msg.Content,
		Metadata:  json.RawMessage(msg.Metadata),
		CreatedAt: msg.CreatedAt,
	}
}

// 将会话实体转换成 API DTO。
func sessionDTO(session model.Session) model.SessionDTO {
	return model.SessionDTO{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		Title:     session.Title,
		Status:    session.Status,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		EndedAt:   session.EndedAt,
	}
}

// 将行程实体转换成 API DTO。
func tripDTO(trip model.Trip) model.TripDTO {
	return model.TripDTO{
		TripID:        trip.TripID,
		UserID:        trip.UserID,
		SessionID:     trip.SessionID,
		Origin:        trip.Origin,
		Destination:   trip.Destination,
		StartDate:     formatDate(trip.StartDate),
		EndDate:       formatDate(trip.EndDate),
		Purpose:       trip.Purpose,
		ItineraryJSON: json.RawMessage(trip.ItineraryJSON),
		RawJSON:       json.RawMessage(trip.RawJSON),
		CreatedAt:     trip.CreatedAt,
		UpdatedAt:     trip.UpdatedAt,
	}
}

// 将消息实体转换成 Redis 短期记忆结构。
func recentFromMessage(msg model.ChatMessage) cache.RecentMessage {
	return cache.RecentMessage{
		MessageID: msg.MessageID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

// 按 Python 记忆逻辑追加偏好并去重。
func appendPreferenceValue(existing, incoming json.RawMessage) (json.RawMessage, error) {
	var currentItems []any
	if len(existing) > 0 && string(existing) != "null" {
		if err := json.Unmarshal(existing, &currentItems); err != nil {
			var scalar any
			if err2 := json.Unmarshal(existing, &scalar); err2 != nil {
				return nil, err
			}
			currentItems = []any{scalar}
		}
	}
	var incomingItems []any
	if err := json.Unmarshal(incoming, &incomingItems); err != nil {
		var scalar any
		if err2 := json.Unmarshal(incoming, &scalar); err2 != nil {
			return nil, err
		}
		incomingItems = []any{scalar}
	}
	seen := map[string]bool{}
	merged := make([]any, 0, len(currentItems)+len(incomingItems))
	for _, item := range append(currentItems, incomingItems...) {
		keyBytes, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		key := string(keyBytes)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, item)
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 规范化偏好更新动作。
func normalizeAction(action string) string {
	switch action {
	case "", "replace":
		return "replace"
	case "append":
		return "append"
	default:
		return "replace"
	}
}

// 校验请求中的 JSON 原始值。
func validateRawJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("json value is empty")
	}
	var v any
	return json.Unmarshal(raw, &v)
}
