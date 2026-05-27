// Package model 定义数据库实体和 HTTP DTO。
package model

import (
	"time"

	"gorm.io/datatypes"
)

// 系统中的业务用户。
type User struct {
	ID          uint64 `gorm:"primaryKey"`
	UserID      string `gorm:"uniqueIndex;size:128;not null"`
	DisplayName string `gorm:"size:128"`
	Source      string `gorm:"size:64;not null;default:local"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 一次用户会话。
type Session struct {
	ID        uint64 `gorm:"primaryKey"`
	SessionID string `gorm:"uniqueIndex;size:128;not null"`
	UserID    string `gorm:"index;size:128;not null"`
	Title     string `gorm:"size:255"`
	Status    string `gorm:"size:32;not null;default:active"`
	CreatedAt time.Time
	UpdatedAt time.Time
	EndedAt   *time.Time
}

// 会话中的用户或助手消息。
type ChatMessage struct {
	ID        uint64 `gorm:"primaryKey"`
	MessageID string `gorm:"uniqueIndex;size:128;not null"`
	SessionID string `gorm:"index:idx_messages_session_created,priority:1;size:128;not null"`
	UserID    string `gorm:"index:idx_messages_user_created,priority:1;size:128;not null"`
	Role      string `gorm:"size:32;not null"`
	Content   string `gorm:"type:mediumtext;not null"`
	Metadata  datatypes.JSON
	CreatedAt time.Time `gorm:"index:idx_messages_session_created,priority:2;index:idx_messages_user_created,priority:2"`
}

// 用户长期偏好，value_json 保存原始 JSON 值。
type Preference struct {
	ID        uint64         `gorm:"primaryKey"`
	UserID    string         `gorm:"uniqueIndex:uk_preferences_user_type,priority:1;size:128;not null"`
	Type      string         `gorm:"uniqueIndex:uk_preferences_user_type,priority:2;size:128;not null"`
	ValueJSON datatypes.JSON `gorm:"column:value_json;type:json;not null"`
	Source    string         `gorm:"size:64;not null;default:agent"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 用户历史行程记录。
type Trip struct {
	ID            uint64 `gorm:"primaryKey"`
	TripID        string `gorm:"uniqueIndex;size:128;not null"`
	UserID        string `gorm:"index:idx_trips_user_created,priority:1;size:128;not null"`
	SessionID     string `gorm:"size:128"`
	Origin        string `gorm:"size:255"`
	Destination   string `gorm:"index;size:255"`
	StartDate     *time.Time
	EndDate       *time.Time
	Purpose       string `gorm:"size:255"`
	ItineraryJSON datatypes.JSON
	RawJSON       datatypes.JSON
	CreatedAt     time.Time `gorm:"index:idx_trips_user_created,priority:2"`
	UpdatedAt     time.Time
}

// 指定行程实体对应 PLAN 中的 trip_history 表。
func (Trip) TableName() string {
	return "trip_history"
}

// 记录一次 Python Agent 调用的请求、结果和错误状态。
type AgentRun struct {
	ID               uint64 `gorm:"primaryKey"`
	RunID            string `gorm:"uniqueIndex;size:128;not null"`
	UserID           string `gorm:"size:128;not null"`
	SessionID        string `gorm:"index:idx_agent_runs_session_created,priority:1;size:128"`
	RequestMessageID string `gorm:"size:128"`
	Status           string `gorm:"index:idx_agent_runs_status_created,priority:1;size:32;not null"`
	RequestJSON      datatypes.JSON
	IntentionJSON    datatypes.JSON
	ScheduleJSON     datatypes.JSON
	ResultJSON       datatypes.JSON
	ErrorCode        string `gorm:"size:128"`
	ErrorMessage     string `gorm:"type:text"`
	DurationMS       *int
	CreatedAt        time.Time `gorm:"index:idx_agent_runs_session_created,priority:2;index:idx_agent_runs_status_created,priority:2"`
}

// 保存从历史记忆迁移来的用户统计信息。
type UserStatistics struct {
	ID                       uint64 `gorm:"primaryKey"`
	UserID                   string `gorm:"uniqueIndex;size:128;not null"`
	TotalTrips               int
	TotalMessages            int
	FrequentDestinationsJSON datatypes.JSON `gorm:"column:frequent_destinations_json;type:json"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// 指定用户统计实体对应 user_statistics 表。
func (UserStatistics) TableName() string {
	return "user_statistics"
}
