// Package config 负责从环境变量加载后端运行配置。
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 保存后端服务启动所需的全部运行配置。
type Config struct {
	AppEnv                    string
	HTTPAddr                  string
	MySQLDSN                  string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	PythonAgentBaseURL        string
	PythonAgentTimeoutSeconds int
	ChatMaxRecentTurns        int
	AdminToken                string
}

// Load 从环境变量读取配置，并为本地开发提供安全默认值。
func Load() Config {
	return Config{
		AppEnv:                    getEnv("APP_ENV", "development"),
		HTTPAddr:                  getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:                  os.Getenv("MYSQL_DSN"),
		RedisAddr:                 os.Getenv("REDIS_ADDR"),
		RedisPassword:             os.Getenv("REDIS_PASSWORD"),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		PythonAgentBaseURL:        strings.TrimRight(os.Getenv("PYTHON_AGENT_BASE_URL"), "/"),
		PythonAgentTimeoutSeconds: getEnvInt("PYTHON_AGENT_TIMEOUT_SECONDS", 120),
		ChatMaxRecentTurns:        getEnvInt("CHAT_MAX_RECENT_TURNS", 10),
		AdminToken:                os.Getenv("ADMIN_TOKEN"),
	}
}

// PythonAgentTimeout 返回调用 Python Agent 的超时时间。
func (c Config) PythonAgentTimeout() time.Duration {
	if c.PythonAgentTimeoutSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(c.PythonAgentTimeoutSeconds) * time.Second
}

// MaxRecentMessages 将“轮数”配置转换为消息条数上限。
func (c Config) MaxRecentMessages() int {
	turns := c.ChatMaxRecentTurns
	if turns <= 0 {
		turns = 10
	}
	return turns * 2
}

// AdminEnabled 判断管理接口是否允许启用。
func (c Config) AdminEnabled() bool {
	return c.AppEnv != "production" || c.AdminToken != ""
}

// getEnv 读取字符串环境变量，未设置时返回默认值。
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt 读取整数环境变量，解析失败时返回默认值。
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
