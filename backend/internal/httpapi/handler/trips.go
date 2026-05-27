// Package handler 实现行程接口处理器。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/httpapi/respond"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/service"
)

// TripHandler 处理用户行程请求。
type TripHandler struct {
	service *service.TripService
}

// NewTripHandler 创建行程 handler。
func NewTripHandler(service *service.TripService) *TripHandler {
	return &TripHandler{service: service}
}

// List 处理用户行程列表查询。
func (h *TripHandler) List(c *gin.Context) {
	limit, offset := pagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
	trips, err := h.service.List(c.Request.Context(), c.Param("user_id"), limit, offset)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, gin.H{"trips": trips})
}

// Create 处理新增行程。
func (h *TripHandler) Create(c *gin.Context) {
	var req model.TripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, service.InvalidArgument("invalid trip request"))
		return
	}
	trip, err := h.service.Create(c.Request.Context(), c.Param("user_id"), req)
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.Created(c, trip)
}

// Get 处理查询单个行程。
func (h *TripHandler) Get(c *gin.Context) {
	trip, err := h.service.Get(c.Request.Context(), c.Param("user_id"), c.Param("trip_id"))
	if err != nil {
		respond.Fail(c, err)
		return
	}
	respond.OK(c, trip)
}

// Delete 处理删除单个行程。
func (h *TripHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("user_id"), c.Param("trip_id")); err != nil {
		respond.Fail(c, err)
		return
	}
	respond.NoContent(c)
}

// pagination 解析 handler 层的分页参数。
func pagination(limitText, offsetText string) (int, int) {
	limit := 20
	offset := 0
	if parsed, err := strconv.Atoi(limitText); err == nil && parsed > 0 {
		limit = parsed
	}
	if parsed, err := strconv.Atoi(offsetText); err == nil && parsed >= 0 {
		offset = parsed
	}
	return limit, offset
}
