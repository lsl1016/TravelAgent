// Package service 实现会话和消息查询业务逻辑。
package service

import (
	"context"

	"travelagent/backend/internal/model"
	"travelagent/backend/internal/repository"
)

// SessionService 负责会话创建、查询和消息读取。
type SessionService struct {
	store *repository.Store
}

// NewSessionService 创建会话服务。
func NewSessionService(store *repository.Store) *SessionService {
	return &SessionService{store: store}
}

// Create 创建用户会话，必要时先创建用户。
func (s *SessionService) Create(ctx context.Context, req model.CreateSessionRequest) (*model.SessionDTO, error) {
	if req.UserID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	if _, err := s.store.EnsureUser(ctx, req.UserID); err != nil {
		return nil, MapError(err)
	}
	session, err := s.store.CreateSession(ctx, model.Session{
		SessionID: newID("sess"),
		UserID:    req.UserID,
		Title:     req.Title,
		Status:    "active",
	})
	if err != nil {
		return nil, MapError(err)
	}
	dto := sessionDTO(*session)
	return &dto, nil
}

// List 分页查询用户会话。
func (s *SessionService) List(ctx context.Context, userID string, limit, offset int) ([]model.SessionDTO, error) {
	if userID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	sessions, err := s.store.ListSessions(ctx, userID, limit, offset)
	if err != nil {
		return nil, MapError(err)
	}
	dtos := make([]model.SessionDTO, 0, len(sessions))
	for _, session := range sessions {
		dtos = append(dtos, sessionDTO(session))
	}
	return dtos, nil
}

// Messages 查询指定会话的消息列表。
func (s *SessionService) Messages(ctx context.Context, sessionID string, limit int, before string) ([]model.MessageDTO, error) {
	if sessionID == "" {
		return nil, InvalidArgument("session_id is required")
	}
	messages, err := s.store.ListMessages(ctx, sessionID, limit, before)
	if err != nil {
		return nil, MapError(err)
	}
	dtos := make([]model.MessageDTO, 0, len(messages))
	for _, msg := range messages {
		dtos = append(dtos, messageDTO(msg))
	}
	return dtos, nil
}

// Delete 关闭或删除指定会话。
func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return InvalidArgument("session_id is required")
	}
	if err := s.store.DeleteSession(ctx, sessionID); err != nil {
		return MapError(err)
	}
	return nil
}
