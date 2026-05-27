// Package handler 实现 HTTP 请求处理器。
package handler

import (
	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/service"
)

// HealthHandler 处理健康检查请求。
type HealthHandler struct {
	service *service.HealthService
}

// NewHealthHandler 创建健康检查 handler。
func NewHealthHandler(service *service.HealthService) *HealthHandler {
	return &HealthHandler{service: service}
}

// Check 处理 GET /health。
func (h *HealthHandler) Check(c *gin.Context) {
	respond.OK(c, h.service.Check(c.Request.Context()))
}
