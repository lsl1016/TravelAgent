// Package handler 实现偏好接口处理器。
package handler

import (
	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/service"
)

// PreferenceHandler 处理用户偏好请求。
type PreferenceHandler struct {
	service *service.PreferenceService
}

// NewPreferenceHandler 创建偏好 handler。
func NewPreferenceHandler(service *service.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{service: service}
}

// List 处理查询用户偏好列表。
func (h *PreferenceHandler) List(c *gin.Context) {
	prefs, err := h.service.List(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, model.PreferenceListResponse{Preferences: prefs})
}

// Put 处理批量更新用户偏好。
func (h *PreferenceHandler) Put(c *gin.Context) {
	var req model.PutPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, service.InvalidArgument("invalid preferences request"))
		return
	}
	prefs, err := h.service.Put(c.Request.Context(), c.Param("user_id"), req.Preferences)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, model.PreferenceListResponse{Preferences: prefs})
}

// Patch 处理单个偏好更新。
func (h *PreferenceHandler) Patch(c *gin.Context) {
	var req model.PatchPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, service.InvalidArgument("invalid preference request"))
		return
	}
	pref, err := h.service.Patch(c.Request.Context(), c.Param("user_id"), c.Param("type"), req.Value, req.Action, req.Source)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, pref)
}

// Delete 处理删除单个偏好。
func (h *PreferenceHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("user_id"), c.Param("type")); err != nil {
		respond.Fail(c, err)
		return
	}
	respond.NoContent(c)
}
