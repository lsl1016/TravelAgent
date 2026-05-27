// Package service 实现历史 JSON 记忆文件到 MySQL 的迁移逻辑。
package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"travelagent/backend/internal/model"
	"travelagent/backend/internal/repository"
)

// MigrationService 负责将历史 JSON 记忆文件迁移到 MySQL。
type MigrationService struct {
	store *repository.Store
}

// NewMigrationService 创建迁移服务。
func NewMigrationService(store *repository.Store) *MigrationService {
	return &MigrationService{store: store}
}

// MigrateMemoryJSON 扫描 data/memory/*.json 并执行 dry-run 或导入。
func (s *MigrationService) MigrateMemoryJSON(ctx context.Context, req model.MigrationRequest) (*model.MigrationReport, error) {
	root := req.Path
	if root == "" {
		root = filepath.Join("data", "memory")
	}
	report := &model.MigrationReport{
		JobID:  newID("mig"),
		DryRun: req.DryRun,
		Errors: []string{},
	}
	files, err := filepath.Glob(filepath.Join(root, "*.json"))
	if err != nil {
		return nil, NewError(http.StatusBadRequest, CodeInvalidArgument, "invalid migration path", err)
	}
	for _, file := range files {
		if err := s.migrateFile(ctx, file, req.DryRun, report); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", file, err))
		}
	}
	if len(report.Errors) > 0 && report.Users == 0 {
		return report, NewError(http.StatusBadRequest, CodeMigrationFailed, "migration failed", nil)
	}
	return report, nil
}

// migrateFile 解析单个用户记忆文件并迁移各类数据。
func (s *MigrationService) migrateFile(ctx context.Context, file string, dryRun bool, report *model.MigrationReport) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	userID := stringValue(doc["user_id"])
	if userID == "" {
		userID = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	}
	report.Users++
	if !dryRun {
		if _, err := s.store.EnsureUser(ctx, userID); err != nil {
			return err
		}
	}
	if err := s.migratePreferences(ctx, userID, doc["preferences"], dryRun, report); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s preferences: %v", userID, err))
	}
	if err := s.migrateMessages(ctx, userID, doc["chat_history"], dryRun, report); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s chat_history: %v", userID, err))
	}
	if err := s.migrateTrips(ctx, userID, doc["trip_history"], dryRun, report); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s trip_history: %v", userID, err))
	}
	if err := s.migrateStatistics(ctx, userID, doc["statistics"], dryRun, report); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s statistics: %v", userID, err))
	}
	return nil
}

// migratePreferences 迁移新旧多种格式的用户偏好。
func (s *MigrationService) migratePreferences(ctx context.Context, userID string, value any, dryRun bool, report *model.MigrationReport) error {
	prefs := normalizePreferences(value)
	for typ, raw := range prefs {
		report.Preferences++
		if dryRun {
			continue
		}
		if _, err := s.store.UpsertPreference(ctx, userID, typ, rawToJSON(raw), "migration"); err != nil {
			return err
		}
	}
	return nil
}

// migrateMessages 迁移历史聊天消息，并按 legacy session 补齐会话。
func (s *MigrationService) migrateMessages(ctx context.Context, userID string, value any, dryRun bool, report *model.MigrationReport) error {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	sessionIDs := map[string]bool{}
	for i, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sessionID := stringValue(obj["session_id"])
		if sessionID == "" {
			sessionID = "legacy_" + userID
		}
		if !sessionIDs[sessionID] {
			report.Sessions++
			sessionIDs[sessionID] = true
			if !dryRun {
				if _, err := s.store.EnsureSession(ctx, sessionID, userID, "legacy import"); err != nil {
					return err
				}
			}
		}
		report.Messages++
		if dryRun {
			continue
		}
		createdAt := parseTimeValue(obj["timestamp"])
		msg := model.ChatMessage{
			MessageID: stableID("legacy_msg", userID, sessionID, fmt.Sprint(i), stringValue(obj["content"])),
			SessionID: sessionID,
			UserID:    userID,
			Role:      stringValue(obj["role"]),
			Content:   stringValue(obj["content"]),
			Metadata:  mustJSON(obj),
			CreatedAt: createdAt,
		}
		if msg.Role == "" {
			msg.Role = "user"
		}
		if _, err := s.store.CreateMessage(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// migrateTrips 迁移历史行程，并保留原始 JSON。
func (s *MigrationService) migrateTrips(ctx context.Context, userID string, value any, dryRun bool, report *model.MigrationReport) error {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	for i, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		report.Trips++
		if dryRun {
			continue
		}
		start, _ := parseDate(stringValue(obj["start_date"]))
		end, _ := parseDate(stringValue(obj["end_date"]))
		tripID := stringValue(obj["trip_id"])
		if tripID == "" {
			tripID = fmt.Sprintf("legacy_trip_%d", i+1)
		}
		_, err := s.store.UpsertTrip(ctx, model.Trip{
			TripID:      tripID,
			UserID:      userID,
			SessionID:   stringValue(obj["session_id"]),
			Origin:      stringValue(obj["origin"]),
			Destination: stringValue(obj["destination"]),
			StartDate:   start,
			EndDate:     end,
			Purpose:     stringValue(obj["purpose"]),
			RawJSON:     mustJSON(obj),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// migrateStatistics 迁移用户统计信息。
func (s *MigrationService) migrateStatistics(ctx context.Context, userID string, value any, dryRun bool, report *model.MigrationReport) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	report.Statistics++
	if dryRun {
		return nil
	}
	stats := model.UserStatistics{
		UserID:                   userID,
		TotalTrips:               int(floatValue(obj["total_trips"])),
		TotalMessages:            int(floatValue(obj["total_messages"])),
		FrequentDestinationsJSON: mustJSON(obj["frequent_destinations"]),
	}
	return s.store.UpsertStatistics(ctx, stats)
}

// normalizePreferences 兼容数组、字典和嵌套 preferences 三种格式。
func normalizePreferences(value any) map[string]json.RawMessage {
	result := map[string]json.RawMessage{}
	switch prefs := value.(type) {
	case []any:
		for _, item := range prefs {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			typ := stringValue(obj["type"])
			if typ == "preferences" {
				for nestedType, nestedRaw := range normalizePreferences(obj["value"]) {
					result[nestedType] = nestedRaw
				}
				continue
			}
			if typ == "" {
				continue
			}
			result[typ] = mustRaw(obj["value"])
		}
	case map[string]any:
		for typ, raw := range prefs {
			result[typ] = mustRaw(raw)
		}
	}
	return result
}

// mustRaw 将任意值转换成 RawMessage。
func mustRaw(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return data
}

// stringValue 宽松地把 JSON 值转换成字符串。
func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

// floatValue 宽松地把 JSON 数字转换成 float64。
func floatValue(value any) float64 {
	if number, ok := value.(float64); ok {
		return number
	}
	return 0
}

// parseTimeValue 兼容历史 JSON 中多种时间格式。
func parseTimeValue(value any) time.Time {
	text := stringValue(value)
	if text == "" {
		return time.Now().UTC()
	}
	layouts := []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}

// stableID 根据历史数据内容生成幂等 ID。
func stableID(prefix string, parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	return prefix + "_" + hex.EncodeToString(hash[:])[:20]
}
