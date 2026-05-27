// Package adapter 隔离 Go 后端与 Python Agent Runtime 的调用协议。
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	CodeAgentUnavailable = "PYTHON_AGENT_UNAVAILABLE"
	CodeAgentTimeout     = "PYTHON_AGENT_TIMEOUT"
	CodeAgentBadResponse = "PYTHON_AGENT_BAD_RESPONSE"
)

type Error struct {
	Code    string
	Message string
	Err     error
}

// Error 返回适配器错误的可读描述。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap 暴露底层错误，便于 errors.As / errors.Is 识别。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PythonAgent 通过 HTTP 调用常驻 Python Agent Worker。
type PythonAgent struct {
	baseURL string
	client  *http.Client
}

// RunRequest 是发送给 Python Agent /run 接口的请求体。
type RunRequest struct {
	UserID    string         `json:"user_id"`
	SessionID string         `json:"session_id"`
	Message   string         `json:"message"`
	Context   map[string]any `json:"context"`
}

// NewPythonAgent 创建 Python Agent HTTP 适配器。
func NewPythonAgent(baseURL string, timeout time.Duration) *PythonAgent {
	return &PythonAgent{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

// Health 检查 Python Agent Worker 是否可用。
func (p *PythonAgent) Health(ctx context.Context) error {
	if p == nil || p.baseURL == "" {
		return &Error{Code: CodeAgentUnavailable, Message: "python agent base url is not configured"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/health", nil)
	if err != nil {
		return &Error{Code: CodeAgentUnavailable, Message: "build python health request failed", Err: err}
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return classifyHTTPError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return &Error{Code: CodeAgentUnavailable, Message: fmt.Sprintf("python agent health returned %d", resp.StatusCode)}
	}
	return nil
}

// Run 调用 Python Agent 执行一次用户消息编排。
func (p *PythonAgent) Run(ctx context.Context, request RunRequest) (map[string]any, error) {
	if p == nil || p.baseURL == "" {
		return nil, &Error{Code: CodeAgentUnavailable, Message: "python agent base url is not configured"}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, &Error{Code: CodeAgentBadResponse, Message: "marshal python request failed", Err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/run", bytes.NewReader(body))
	if err != nil {
		return nil, &Error{Code: CodeAgentUnavailable, Message: "build python run request failed", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, classifyHTTPError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusRequestTimeout {
		return nil, &Error{Code: CodeAgentTimeout, Message: "python agent call timed out"}
	}
	if resp.StatusCode >= 500 {
		return nil, &Error{Code: CodeAgentUnavailable, Message: fmt.Sprintf("python agent returned %d", resp.StatusCode)}
	}
	if resp.StatusCode >= 400 {
		return nil, &Error{Code: CodeAgentBadResponse, Message: fmt.Sprintf("python agent rejected request with %d", resp.StatusCode)}
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &Error{Code: CodeAgentBadResponse, Message: "python agent returned invalid json", Err: err}
	}
	if result == nil {
		return nil, &Error{Code: CodeAgentBadResponse, Message: "python agent returned empty json"}
	}
	return result, nil
}

// classifyHTTPError 将网络错误映射为稳定的业务错误码。
func classifyHTTPError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: CodeAgentTimeout, Message: "python agent call timed out", Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Error{Code: CodeAgentTimeout, Message: "python agent call timed out", Err: err}
	}
	return &Error{Code: CodeAgentUnavailable, Message: "python agent is unavailable", Err: err}
}
