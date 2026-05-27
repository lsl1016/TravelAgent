// Package service 实现聊天主链路编排。
package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"travelagent/backend/internal/adapter"
	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/repository"
)

// ChatService 负责聊天主链路的业务编排。
type ChatService struct {
	store             *repository.Store
	cache             *cache.Cache
	agent             *adapter.PythonAgent
	maxRecentMessages int
}

// NewChatService 创建聊天服务。
func NewChatService(store *repository.Store, cache *cache.Cache, agent *adapter.PythonAgent, maxRecentMessages int) *ChatService {
	if maxRecentMessages <= 0 {
		maxRecentMessages = 20
	}
	return &ChatService{store: store, cache: cache, agent: agent, maxRecentMessages: maxRecentMessages}
}

// Chat 串联用户/会话、消息持久化、上下文读取、Python Agent 调用和短期记忆刷新。
func (s *ChatService) Chat(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	req.UserID = strings.TrimSpace(req.UserID)
	req.Message = strings.TrimSpace(req.Message)
	if req.UserID == "" || req.Message == "" {
		return nil, InvalidArgument("user_id and message are required")
	}
	if req.SessionID == "" {
		req.SessionID = "sess_" + uuid.NewString()
	}
	if _, err := s.store.EnsureUser(ctx, req.UserID); err != nil {
		return nil, MapError(err)
	}
	if _, err := s.store.EnsureSession(ctx, req.SessionID, req.UserID, titleFromMessage(req.Message)); err != nil {
		return nil, MapError(err)
	}

	userMsg := model.ChatMessage{
		MessageID: newID("msg"),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "user",
		Content:   req.Message,
		Metadata:  mustJSON(req.Metadata),
	}
	if _, err := s.store.CreateMessage(ctx, userMsg); err != nil {
		return nil, MapError(err)
	}

	contextPayload := s.buildAgentContext(ctx, req.UserID, req.SessionID)
	runID := newID("run")
	agentReq := adapter.RunRequest{
		UserID:    req.UserID,
		SessionID: req.SessionID,
		Message:   req.Message,
		Context:   contextPayload,
	}
	started := time.Now()
	agentResult, agentErr := s.agent.Run(ctx, agentReq)
	duration := int(time.Since(started).Milliseconds())
	run := model.AgentRun{
		RunID:            runID,
		UserID:           req.UserID,
		SessionID:        req.SessionID,
		RequestMessageID: userMsg.MessageID,
		RequestJSON:      mustJSON(agentReq),
		DurationMS:       &duration,
	}
	if agentErr != nil {
		appErr := MapError(agentErr)
		run.Status = statusFromAgentError(appErr.Code)
		run.ErrorCode = appErr.Code
		run.ErrorMessage = appErr.Message
		_, _ = s.store.UpsertAgentRun(ctx, run)
		_ = s.cache.SetAgentRunStatus(ctx, runID, map[string]any{"status": run.Status, "error_code": run.ErrorCode})
		return nil, appErr
	}

	run.Status = statusFromResult(agentResult)
	run.ResultJSON = mustJSON(agentResult)
	run.IntentionJSON = extractJSON(agentResult, "intention")
	run.ScheduleJSON = extractJSON(agentResult, "agent_schedule")
	if _, err := s.store.UpsertAgentRun(ctx, run); err != nil {
		return nil, MapError(err)
	}

	assistantContent := assistantContent(agentResult)
	assistantMsg := model.ChatMessage{
		MessageID: newID("msg"),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "assistant",
		Content:   assistantContent,
		Metadata:  datatypes.JSON([]byte(`{"source":"python_agent"}`)),
	}
	if _, err := s.store.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, MapError(err)
	}
	_ = s.cache.AppendRecentMessages(ctx, req.SessionID, recentFromMessage(userMsg), recentFromMessage(assistantMsg))
	_ = s.cache.SetAgentRunStatus(ctx, runID, map[string]any{"status": run.Status})

	return &model.ChatResponse{
		SessionID:   req.SessionID,
		MessageID:   assistantMsg.MessageID,
		AgentRunID:  runID,
		AgentResult: agentResult,
	}, nil
}

// buildAgentContext 组装发送给 Python Agent 的近期消息、偏好和历史行程上下文。
func (s *ChatService) buildAgentContext(ctx context.Context, userID, sessionID string) map[string]any {
	recent, err := s.cache.GetRecentMessages(ctx, sessionID)
	if err != nil || len(recent) == 0 {
		if messages, fallbackErr := s.store.ListMessages(ctx, sessionID, s.maxRecentMessages, ""); fallbackErr == nil {
			recent = make([]cache.RecentMessage, 0, len(messages))
			for _, msg := range messages {
				recent = append(recent, recentFromMessage(msg))
			}
			_ = s.cache.ReplaceRecentMessages(ctx, sessionID, recent)
		}
	}
	preferences, err := s.cache.GetPreferences(ctx, userID)
	if err != nil || preferences == nil {
		if prefs, fallbackErr := s.store.GetPreferences(ctx, userID); fallbackErr == nil {
			preferences = preferenceMap(prefs)
			_ = s.cache.SetPreferences(ctx, userID, preferences)
		}
	}
	trips, _ := s.store.ListTrips(ctx, userID, 20, 0)
	tripDTOs := make([]model.TripDTO, 0, len(trips))
	for _, trip := range trips {
		tripDTOs = append(tripDTOs, tripDTO(trip))
	}
	return map[string]any{
		"recent_messages": recent,
		"preferences":     preferences,
		"trip_history":    tripDTOs,
	}
}

// titleFromMessage 从首条消息生成默认会话标题。
func titleFromMessage(message string) string {
	runes := []rune(message)
	if len(runes) > 30 {
		return string(runes[:30])
	}
	return message
}

// statusFromResult 从 Agent 结果中提取执行状态。
func statusFromResult(result map[string]any) string {
	if status, ok := result["status"].(string); ok && status != "" {
		return status
	}
	return "completed"
}

// statusFromAgentError 将 Agent 错误码转换为执行记录状态。
func statusFromAgentError(code string) string {
	if code == CodePythonAgentTimeout {
		return "timeout"
	}
	return "error"
}

// extractJSON 从 Agent 结果中抽取指定字段并保存为 JSON。
func extractJSON(result map[string]any, key string) datatypes.JSON {
	value, ok := result[key]
	if !ok {
		return nil
	}
	return mustJSON(value)
}

// assistantContent 从 Agent 结果中提取助手消息正文。
func assistantContent(result map[string]any) string {
	if text, ok := result["message"].(string); ok && text != "" {
		return text
	}
	if text, ok := result["content"].(string); ok && text != "" {
		return text
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ValidatePythonAvailable 主动检查 Python Agent 可用性。
func (s *ChatService) ValidatePythonAvailable(ctx context.Context) error {
	if err := s.agent.Health(ctx); err != nil {
		return NewError(http.StatusBadGateway, CodePythonAgentUnavailable, "Python Agent is unavailable", err)
	}
	return nil
}
