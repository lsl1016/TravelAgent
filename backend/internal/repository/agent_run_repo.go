// Package repository 提供 Agent 执行记录表访问方法。
package repository

import (
	"context"

	"gorm.io/gorm/clause"

	"travelagent/backend/internal/model"
)

// =创建或更新一次 Agent 执行记录。
func (s *Store) UpsertAgentRun(ctx context.Context, run model.AgentRun) (*model.AgentRun, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now()
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "request_json", "intention_json", "schedule_json", "result_json",
			"error_code", "error_message", "duration_ms",
		}),
	}).Create(&run).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("run_id = ?", run.RunID).First(&run).Error
	return &run, err
}
