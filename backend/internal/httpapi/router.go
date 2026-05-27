// Package httpapi 装配 Gin 路由和各业务 handler。
package httpapi

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"travelagent/backend/internal/config"
	"travelagent/backend/internal/httpapi/handler"
	"travelagent/backend/internal/httpapi/middleware"
	"travelagent/backend/internal/service"
)

// 聚合路由层需要注入的业务服务。
type Services struct {
	Health     *service.HealthService
	Chat       *service.ChatService
	Preference *service.PreferenceService
	Trip       *service.TripService
	Session    *service.SessionService
	Migration  *service.MigrationService
}

// 创建 HTTP 路由，并注册健康检查、业务 API 和管理 API。
func NewRouter(cfg config.Config, services Services, log *slog.Logger) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestID(), middleware.Logger(log))

	healthHandler := handler.NewHealthHandler(services.Health)
	chatHandler := handler.NewChatHandler(services.Chat)
	preferenceHandler := handler.NewPreferenceHandler(services.Preference)
	tripHandler := handler.NewTripHandler(services.Trip)
	sessionHandler := handler.NewSessionHandler(services.Session)
	adminHandler := handler.NewAdminHandler(services.Migration)

	router.GET("/health", healthHandler.Check)

	v1 := router.Group("/api/v1")
	v1.POST("/chat", chatHandler.Chat)

	v1.POST("/sessions", sessionHandler.Create)
	v1.GET("/users/:user_id/sessions", sessionHandler.List)
	v1.GET("/sessions/:session_id/messages", sessionHandler.Messages)
	v1.DELETE("/sessions/:session_id", sessionHandler.Delete)

	v1.GET("/users/:user_id/preferences", preferenceHandler.List)
	v1.PUT("/users/:user_id/preferences", preferenceHandler.Put)
	v1.PATCH("/users/:user_id/preferences/:type", preferenceHandler.Patch)
	v1.DELETE("/users/:user_id/preferences/:type", preferenceHandler.Delete)

	v1.GET("/users/:user_id/trips", tripHandler.List)
	v1.POST("/users/:user_id/trips", tripHandler.Create)
	v1.GET("/users/:user_id/trips/:trip_id", tripHandler.Get)
	v1.DELETE("/users/:user_id/trips/:trip_id", tripHandler.Delete)

	admin := v1.Group("/admin", middleware.Admin(cfg.AdminEnabled(), cfg.AdminToken))
	admin.POST("/migrate-memory-json", adminHandler.MigrateMemoryJSON)
	admin.GET("/migrate-memory-json/status/:job_id", adminHandler.MigrationStatus)

	return router
}
