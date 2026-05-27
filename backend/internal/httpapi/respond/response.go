// Package respond 统一封装 HTTP API 的成功和错误响应。
package respond

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/service"
)

// Response 是 PLAN 中约定的通用 JSON 响应信封。
type Response struct {
	RequestID string            `json:"request_id"`
	Status    string            `json:"status"`
	Data      any               `json:"data"`
	Error     *service.AppError `json:"error"`
}

// OK 返回 200 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		RequestID: RequestID(c),
		Status:    "success",
		Data:      data,
		Error:     nil,
	})
}

// Created 返回 201 创建成功响应。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		RequestID: RequestID(c),
		Status:    "success",
		Data:      data,
		Error:     nil,
	})
}

// NoContent 返回删除成功的统一响应。
func NoContent(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		RequestID: RequestID(c),
		Status:    "success",
		Data:      gin.H{"deleted": true},
		Error:     nil,
	})
}

// Fail 将服务层错误转换为统一错误响应。
func Fail(c *gin.Context, err error) {
	appErr := service.MapError(err)
	status := appErr.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	c.JSON(status, Response{
		RequestID: RequestID(c),
		Status:    "error",
		Data:      nil,
		Error:     appErr,
	})
}

// RequestID 从 Gin 上下文中读取中间件生成的请求 ID。
func RequestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
