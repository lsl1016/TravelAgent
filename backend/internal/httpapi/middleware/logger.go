// Package middleware 提供结构化访问日志中间件。
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 记录请求 ID、路径、状态码和耗时等访问日志字段。
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		log.Info("http_request",
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status_code", c.Writer.Status(),
			"duration_ms", time.Since(started).Milliseconds(),
			"user_id", c.Param("user_id"),
			"session_id", c.Param("session_id"),
		)
	}
}
