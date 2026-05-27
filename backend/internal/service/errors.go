// Package service 承载后端业务编排和错误映射。
package service

import (
	"errors"
	"net/http"

	"gorm.io/gorm"

	"travelagent/backend/internal/adapter"
	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/repository"
)

const (
	CodeInvalidArgument        = "INVALID_ARGUMENT"
	CodeUserNotFound           = "USER_NOT_FOUND"
	CodeSessionNotFound        = "SESSION_NOT_FOUND"
	CodeMySQLUnavailable       = "MYSQL_UNAVAILABLE"
	CodeRedisUnavailable       = "REDIS_UNAVAILABLE"
	CodePythonAgentUnavailable = "PYTHON_AGENT_UNAVAILABLE"
	CodePythonAgentTimeout     = "PYTHON_AGENT_TIMEOUT"
	CodePythonAgentBadResponse = "PYTHON_AGENT_BAD_RESPONSE"
	CodeMigrationFailed        = "MIGRATION_FAILED"
	CodeInternalError          = "INTERNAL_ERROR"
)

// HTTP 层可以直接序列化的业务错误。
type AppError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	HTTPStatus int            `json:"-"`
	Err        error          `json:"-"`
}

// 返回业务错误的可读描述。
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// 构造带 HTTP 状态码和业务错误码的错误。
func NewError(status int, code, message string, err error) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message, Err: err}
}

// 构造参数错误。
func InvalidArgument(message string) *AppError {
	return NewError(http.StatusBadRequest, CodeInvalidArgument, message, nil)
}

// 将底层错误统一映射为 API 错误模型。
func MapError(err error) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	if errors.Is(err, repository.ErrUnavailable) {
		return NewError(http.StatusServiceUnavailable, CodeMySQLUnavailable, "MySQL is unavailable", err)
	}
	if errors.Is(err, cache.ErrUnavailable) {
		return NewError(http.StatusServiceUnavailable, CodeRedisUnavailable, "Redis is unavailable", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewError(http.StatusNotFound, CodeSessionNotFound, "resource not found", err)
	}
	var agentErr *adapter.Error
	if errors.As(err, &agentErr) {
		switch agentErr.Code {
		case adapter.CodeAgentTimeout:
			return NewError(http.StatusGatewayTimeout, CodePythonAgentTimeout, "Python Agent call timed out", err)
		case adapter.CodeAgentBadResponse:
			return NewError(http.StatusBadGateway, CodePythonAgentBadResponse, "Python Agent returned bad response", err)
		default:
			return NewError(http.StatusBadGateway, CodePythonAgentUnavailable, "Python Agent is unavailable", err)
		}
	}
	return NewError(http.StatusInternalServerError, CodeInternalError, "internal error", err)
}
