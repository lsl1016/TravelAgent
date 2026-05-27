// Package middleware 提供管理接口保护中间件。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Admin 根据环境和可选 token 控制管理接口访问。
func Admin(enabled bool, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "admin api disabled"})
			return
		}
		if token != "" && c.GetHeader("X-Admin-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}
		c.Next()
	}
}
