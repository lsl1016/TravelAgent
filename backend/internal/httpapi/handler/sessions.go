// Package handler 实现会话接口处理器。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/service"
)

// SessionHandler 处理会话和消息请求。
type SessionHandler struct {
	service *service.SessionService
}

// NewSessionHandler 创建会话 handler。
func NewSessionHandler(service *service.SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

// Create 处理创建会话。
func (h *SessionHandler) Create(c *gin.Context) {
	var req model.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, service.InvalidArgument("invalid session request"))
		return
	}
	session, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.Created(c, session)
}

// List 处理用户会话列表查询。
func (h *SessionHandler) List(c *gin.Context) {
	limit, offset := pagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
	sessions, err := h.service.List(c.Request.Context(), c.Param("user_id"), limit, offset)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, gin.H{"sessions": sessions})
}

// Messages 处理会话消息查询。
func (h *SessionHandler) Messages(c *gin.Context) {
	limit := 50
	if parsed, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && parsed > 0 {
		limit = parsed
	}
	messages, err := h.service.Messages(c.Request.Context(), c.Param("session_id"), limit, c.Query("before"))
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, gin.H{"messages": messages})
}

// Delete 处理删除会话。
func (h *SessionHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("session_id")); err != nil {
		respond.Fail(c, err)
		return
	}
	respond.NoContent(c)
}
