package main

import (
	"log/slog"
	"os"

	"travelagent/backend/internal/adapter"
	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/config"
	"travelagent/backend/internal/httpapi"
	"travelagent/backend/internal/repository"
	"travelagent/backend/internal/service"
)

// main 装配配置、数据库、缓存、Agent 适配器和 Gin 路由，并启动 HTTP 服务。
func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	store, err := repository.Open(cfg.MySQLDSN)
	if err != nil {
		log.Error("open_mysql_failed", "error", err)
		os.Exit(1)
	}
	redisCache := cache.Open(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	agent := adapter.NewPythonAgent(cfg.PythonAgentBaseURL, cfg.PythonAgentTimeout())

	services := httpapi.Services{
		Health:     service.NewHealthService(store, redisCache, agent),
		Chat:       service.NewChatService(store, redisCache, agent, cfg.MaxRecentMessages()),
		Preference: service.NewPreferenceService(store, redisCache),
		Trip:       service.NewTripService(store, redisCache),
		Session:    service.NewSessionService(store),
		Migration:  service.NewMigrationService(store),
	}
	router := httpapi.NewRouter(cfg, services, log)

	log.Info("server_starting", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Error("server_stopped", "error", err)
		os.Exit(1)
	}
}
