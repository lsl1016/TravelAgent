# TravelAgent 后端模块

`backend/` 是 TravelAgent 的 Go 后端服务模块，负责把现有 Python Agent 能力包装成稳定的 HTTP API，并提供用户、会话、消息、偏好、行程、Agent 执行记录和历史 JSON 记忆迁移等后端工程能力。

当前实现以 `Gin + GORM + MySQL + Redis` 为基础：

- Gin：提供 REST API。
- GORM + MySQL：持久化用户、会话、消息、偏好、行程、Agent 执行记录和用户统计。
- Redis：缓存短期会话记忆、用户偏好、长期摘要和 Agent 运行状态。
- Python Agent Adapter：通过 HTTP 调用常驻 Python Agent Worker。

## 目录结构

```text
backend/
  cmd/server/                 # HTTP 服务入口
  internal/
    adapter/                  # Python Agent HTTP 适配器
    cache/                    # Redis 缓存与短期记忆
    config/                   # 环境变量配置加载
    httpapi/                  # Gin 路由、handler、中间件和响应封装
      handler/                # 各 REST API handler
      middleware/             # 请求 ID、日志、管理接口保护
      respond/                # 统一响应格式
    model/                    # GORM 实体与 HTTP DTO
    repository/               # MySQL repository
    service/                  # 业务编排层
  migrations/                 # MySQL DDL
  .env.example                # 本地配置示例
  go.mod
  go.sum
```

## 分层约定

- `handler` 只处理 HTTP 入参、状态码和响应格式。
- `service` 编排业务流程，例如聊天主链路、偏好 append/replace、迁移逻辑。
- `repository` 只负责 MySQL 读写。
- `cache` 只负责 Redis key、TTL、短期记忆窗口和缓存失效。
- `adapter` 只负责调用 Python Agent，不泄漏具体通信细节到业务层。
- `model` 保存数据库实体和 API DTO。

## 配置

复制 `.env.example` 后按本地环境设置变量：

```env
APP_ENV=development
HTTP_ADDR=:8080
MYSQL_DSN=user:pass@tcp(127.0.0.1:3306)/travel_agent?parseTime=true&charset=utf8mb4&loc=Local
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0
PYTHON_AGENT_BASE_URL=http://127.0.0.1:8090
PYTHON_AGENT_TIMEOUT_SECONDS=120
CHAT_MAX_RECENT_TURNS=10
ADMIN_TOKEN=
```

说明：

- `MYSQL_DSN` 为空时，涉及 MySQL 的接口会返回不可用状态。
- `REDIS_ADDR` 为空时，缓存能力降级，聊天链路会尽量从 MySQL 回源。
- `PYTHON_AGENT_BASE_URL` 指向 Python Agent Worker，当前约定接口为 `GET /health` 和 `POST /run`。
- `APP_ENV=production` 时，管理接口需要配置 `ADMIN_TOKEN` 并通过 `X-Admin-Token` 访问。

## 数据库初始化

先创建 MySQL 数据库，然后执行：

```bash
mysql -u user -p travel_agent < migrations/001_init.sql
```

DDL 包含以下表：

- `users`
- `sessions`
- `chat_messages`
- `preferences`
- `trip_history`
- `agent_runs`
- `user_statistics`

## 启动服务

在 `backend/` 目录下启动：

```bash
go run ./cmd/server
```

服务默认监听 `:8080`，可以通过 `HTTP_ADDR` 修改。

健康检查：

```bash
curl http://127.0.0.1:8080/health
```

响应会展示 Go 服务、MySQL、Redis 和 Python Agent 的组件状态。

## 主要 API

### 健康检查

```http
GET /health
```

### 聊天

```http
POST /api/v1/chat
```

请求示例：

```json
{
  "user_id": "default_user",
  "session_id": "c83a3b93",
  "message": "我想从北京去杭州出差一周",
  "metadata": {
    "source": "web"
  }
}
```

处理流程：

1. 确保用户和会话存在。
2. 写入用户消息。
3. 从 Redis 或 MySQL 读取最近上下文。
4. 读取用户偏好和历史行程。
5. 调用 Python Agent `/run`。
6. 写入助手消息和 `agent_runs`。
7. 刷新 Redis 短期记忆。

### 用户偏好

```http
GET    /api/v1/users/:user_id/preferences
PUT    /api/v1/users/:user_id/preferences
PATCH  /api/v1/users/:user_id/preferences/:type
DELETE /api/v1/users/:user_id/preferences/:type
```

`PUT` 示例：

```json
{
  "preferences": [
    {
      "type": "hotel_brands",
      "value": ["汉庭", "如家"],
      "action": "replace"
    },
    {
      "type": "airlines",
      "value": "中国国航",
      "action": "append"
    }
  ]
}
```

`append` 会在已有偏好基础上追加并去重，`replace` 会覆盖原值。

### 行程

```http
GET    /api/v1/users/:user_id/trips?limit=20&offset=0
POST   /api/v1/users/:user_id/trips
GET    /api/v1/users/:user_id/trips/:trip_id
DELETE /api/v1/users/:user_id/trips/:trip_id
```

### 会话与消息

```http
POST   /api/v1/sessions
GET    /api/v1/users/:user_id/sessions
GET    /api/v1/sessions/:session_id/messages?limit=50&before=msg_id
DELETE /api/v1/sessions/:session_id
```

### 管理与迁移

```http
POST /api/v1/admin/migrate-memory-json
GET  /api/v1/admin/migrate-memory-json/status/:job_id
```

迁移接口默认扫描 `data/memory/*.json`，也可以通过请求体指定路径：

```json
{
  "path": "../data/memory",
  "dry_run": true
}
```

迁移支持：

- `preferences` 数组格式。
- `preferences` 字典格式。
- 嵌套 `{ "type": "preferences", "value": [...] }` 格式。
- 缺失 `session_id` 的历史消息会归入 `legacy_{user_id}`。
- 缺失 `trip_id` 的历史行程会生成 `legacy_trip_{n}`。
- 重复执行通过唯一键和稳定 ID 尽量保持幂等。

## 统一响应格式

成功响应：

```json
{
  "request_id": "req_20260527_xxx",
  "status": "success",
  "data": {},
  "error": null
}
```

错误响应：

```json
{
  "request_id": "req_20260527_xxx",
  "status": "error",
  "data": null,
  "error": {
    "code": "PYTHON_AGENT_TIMEOUT",
    "message": "Python Agent call timed out"
  }
}
```

常见错误码：

- `INVALID_ARGUMENT`
- `MYSQL_UNAVAILABLE`
- `REDIS_UNAVAILABLE`
- `PYTHON_AGENT_UNAVAILABLE`
- `PYTHON_AGENT_TIMEOUT`
- `PYTHON_AGENT_BAD_RESPONSE`
- `MIGRATION_FAILED`
- `INTERNAL_ERROR`

## 当前验证方式

本模块可以用以下命令做编译级检查：

```bash
go list ./...
go build ./...
```

注意：如果本地环境变量、MySQL、Redis 或 Python Agent 未准备好，不建议直接执行集成测试。当前实现已经避免把 Redis 作为聊天链路的硬依赖，Redis 不可用时会尝试从 MySQL 回源上下文。
