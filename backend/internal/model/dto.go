// Package model 定义 HTTP 请求和响应的数据结构。
package model

import (
	"encoding/json"
	"time"
)

type ChatRequest struct {
	UserID    string                 `json:"user_id" binding:"required"`
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message" binding:"required"`
	Metadata  map[string]any         `json:"metadata"`
	ClientID  string                 `json:"client_message_id"`
	Extra     map[string]interface{} `json:"-"`
}

// =聊天接口返回给调用方的核心结果。
type ChatResponse struct {
	SessionID   string         `json:"session_id"`
	MessageID   string         `json:"message_id"`
	AgentRunID  string         `json:"agent_run_id"`
	AgentResult map[string]any `json:"agent_result"`
}

// 表示单条用户偏好及其更新动作。
type PreferenceDTO struct {
	Type   string          `json:"type" binding:"required"`
	Value  json.RawMessage `json:"value" binding:"required"`
	Action string          `json:"action"`
	Source string          `json:"source"`
}

// 包装偏好列表响应。
type PreferenceListResponse struct {
	Preferences []PreferenceDTO `json:"preferences"`
}

// 批量更新偏好的请求体。
type PutPreferencesRequest struct {
	Preferences []PreferenceDTO `json:"preferences" binding:"required"`
}

// 单个偏好的增量更新请求体。
type PatchPreferenceRequest struct {
	Value  json.RawMessage `json:"value" binding:"required"`
	Action string          `json:"action"`
	Source string          `json:"source"`
}

// 创建或导入行程的请求体。
type TripRequest struct {
	TripID        string          `json:"trip_id"`
	SessionID     string          `json:"session_id"`
	Origin        string          `json:"origin"`
	Destination   string          `json:"destination"`
	StartDate     string          `json:"start_date"`
	EndDate       string          `json:"end_date"`
	Purpose       string          `json:"purpose"`
	ItineraryJSON json.RawMessage `json:"itinerary_json"`
	RawJSON       json.RawMessage `json:"raw_json"`
}

// 对外返回的行程结构。
type TripDTO struct {
	TripID        string          `json:"trip_id"`
	UserID        string          `json:"user_id"`
	SessionID     string          `json:"session_id,omitempty"`
	Origin        string          `json:"origin,omitempty"`
	Destination   string          `json:"destination,omitempty"`
	StartDate     string          `json:"start_date,omitempty"`
	EndDate       string          `json:"end_date,omitempty"`
	Purpose       string          `json:"purpose,omitempty"`
	ItineraryJSON json.RawMessage `json:"itinerary_json,omitempty"`
	RawJSON       json.RawMessage `json:"raw_json,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// 创建会话的请求体。
type CreateSessionRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Title  string `json:"title"`
}

// 对外返回的会话结构。
type SessionDTO struct {
	SessionID string     `json:"session_id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// 对外返回的聊天消息结构。
type MessageDTO struct {
	MessageID string          `json:"message_id"`
	SessionID string          `json:"session_id"`
	UserID    string          `json:"user_id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// JSON 记忆迁移请求。
type MigrationRequest struct {
	Path   string `json:"path"`
	DryRun bool   `json:"dry_run"`
}

// 汇总一次迁移任务的统计和错误。
type MigrationReport struct {
	JobID       string   `json:"job_id"`
	DryRun      bool     `json:"dry_run"`
	Users       int      `json:"users"`
	Sessions    int      `json:"sessions"`
	Messages    int      `json:"messages"`
	Preferences int      `json:"preferences"`
	Trips       int      `json:"trips"`
	Statistics  int      `json:"statistics"`
	Errors      []string `json:"errors"`
}
