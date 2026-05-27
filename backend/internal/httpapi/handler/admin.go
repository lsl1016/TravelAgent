// Package handler 实现管理接口处理器。
package handler

import (
	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/service"
)

// AdminHandler 处理管理和迁移请求。
type AdminHandler struct {
	migration *service.MigrationService
	jobs      map[string]*model.MigrationReport
}

// NewAdminHandler 创建管理接口 handler。
func NewAdminHandler(migration *service.MigrationService) *AdminHandler {
	return &AdminHandler{migration: migration, jobs: map[string]*model.MigrationReport{}}
}

// MigrateMemoryJSON 处理 JSON 记忆迁移请求。
func (h *AdminHandler) MigrateMemoryJSON(c *gin.Context) {
	var req model.MigrationRequest
	_ = c.ShouldBindJSON(&req)
	report, err := h.migration.MigrateMemoryJSON(c.Request.Context(), req)
	if report != nil {
		h.jobs[report.JobID] = report
	}
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, report)
}

// MigrationStatus 查询当前进程内保存的迁移报告。
func (h *AdminHandler) MigrationStatus(c *gin.Context) {
	if report, ok := h.jobs[c.Param("job_id")]; ok {
		respond.OK(c, report)
		return
	}
	respond.OK(c, gin.H{"status": "not_found"})
}
