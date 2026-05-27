// Package handler 实现聊天接口处理器。
package handler

import (
	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/service"
)

// ChatHandler 处理聊天请求。
type ChatHandler struct {
	service *service.ChatService
}

// NewChatHandler 创建聊天 handler。
func NewChatHandler(service *service.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

// Chat 处理 POST /api/v1/chat。
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, service.InvalidArgument("invalid chat request"))
		return
	}
	resp, err := h.service.Chat(c.Request.Context(), req)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, resp)
}
