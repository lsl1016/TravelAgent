// 提供健康检查服务。
package service

import (
	"context"
	"time"

	"travelagent/backend/internal/adapter"
	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/repository"
)

// 聚合外部依赖的健康状态。
type HealthService struct {
	store *repository.Store
	cache *cache.Cache
	agent *adapter.PythonAgent
}

// 创建健康检查服务。
func NewHealthService(store *repository.Store, cache *cache.Cache, agent *adapter.PythonAgent) *HealthService {
	return &HealthService{store: store, cache: cache, agent: agent}
}

// 聚合 Go、MySQL、Redis 和 Python Agent 的健康状态。
func (s *HealthService) Check(ctx context.Context) map[string]any {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	components := map[string]string{
		"go":           "ok",
		"mysql":        "ok",
		"redis":        "ok",
		"python_agent": "ok",
	}
	status := "ok"
	if err := s.store.Ping(ctx); err != nil {
		components["mysql"] = "down"
		status = "down"
	}
	if err := s.cache.Ping(ctx); err != nil {
		components["redis"] = "degraded"
		if status == "ok" {
			status = "degraded"
		}
	}
	if err := s.agent.Health(ctx); err != nil {
		components["python_agent"] = "degraded"
		if status == "ok" {
			status = "degraded"
		}
	}
	return map[string]any{
		"status":     status,
		"components": components,
	}
}
