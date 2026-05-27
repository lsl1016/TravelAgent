# Go 后端模块完善执行计划

## 摘要

项目当前是 Python CLI + AgentScope 多智能体架构。长期记忆存储在 `data/memory/{user_id}.json`，短期记忆存储在进程内列表中；README 中提到的 MySQL、Redis、Gin 属于生产化方向，当前代码尚未落地。

本计划以 **Go + Gin + MySQL + Redis** 完善后端服务能力，目标是把现有 CLI 智能体能力包装成稳定的服务化接口，并把记忆系统从本地 JSON 文件逐步迁移到数据库和缓存。第一阶段以文档和设计为主，不直接重写 AgentScope 子智能体；后续实现采用渐进式方式：Go 后端负责 HTTP API、用户会话、持久化、缓存、健康检查和运维边界，Python 继续负责意图识别、编排和 Skill 执行。

## 目标与非目标

### 目标

- 建立独立 Go 后端服务目录 `backend/`，对外提供 REST API。
- 使用 MySQL 承载用户、会话、聊天记录、偏好、行程历史、Agent 执行记录等长期数据。
- 使用 Redis 承载短期会话记忆、热点偏好、长期摘要、限流和健康状态。
- 为现有 Python Agent 编排链路提供稳定调用入口，让 Go 后端可以通过 HTTP 或子进程/RPC 方式调用。
- 提供 JSON 记忆文件到 MySQL 的迁移方案，兼容当前 `preferences` 新旧格式。
- 明确分阶段实施、测试范围、验收标准和风险处理。

### 非目标

- 第一阶段不重写 `agents/`、`context/`、`.claude/skills/` 中的 Python 智能体逻辑。
- 第一阶段不把 RAG/Milvus、DDGS 搜索、BGE embedding 迁移到 Go。
- 第一阶段不建设完整 Web 前端，只保证后端接口可供后续 Web 或 CLI 调用。
- 第一阶段不引入复杂微服务拆分，避免过早增加部署成本。

## 当前系统现状

### 关键模块

- `cli.py`：当前唯一交互入口，负责初始化模型、记忆管理器、意图识别智能体、协调器，并处理用户自然语言输入。
- `agents/intention_agent.py`：负责意图识别、Query 改写、生成 `agent_schedule`。
- `agents/orchestration_agent.py`：按优先级调度子智能体，同优先级并行执行，聚合结果并更新长期记忆。
- `context/short_term_memory.py`：进程内短期记忆，最多保留 10 轮对话。
- `context/long_term_memory.py`：JSON 文件长期记忆，包含 `preferences`、`chat_history`、`trip_history`、`statistics`。
- `context/memory_manager.py`：统一管理短期记忆和长期记忆，并生成供 Agent 使用的上下文。
- `data/memory/*.json`：现有用户记忆文件，是后续迁移 MySQL 的来源。

### 当前长期记忆 JSON 结构

```json
{
  "user_id": "default_user",
  "created_at": "2026-03-09T22:12:43.711102",
  "updated_at": "2026-03-09T22:28:02.103226",
  "preferences": [
    { "type": "hotel_brands", "value": ["汉庭", "如家"] }
  ],
  "chat_history": [
    {
      "role": "user",
      "content": "我想去杭州玩两天",
      "timestamp": "2026-03-09T22:13:24.041465",
      "session_id": "c83a3b93"
    }
  ],
  "trip_history": [
    {
      "trip_id": "trip_1",
      "timestamp": "2026-03-09T22:13:39.368926",
      "origin": "北京",
      "destination": "杭州",
      "start_date": "2026-03-11",
      "end_date": "2026-03-18",
      "purpose": "出差"
    }
  ],
  "statistics": {
    "total_trips": 2,
    "total_messages": 6,
    "frequent_destinations": {
      "杭州": 2
    }
  }
}
```

## 总体架构

```text
Client / Web / CLI
        |
        v
Go Gin API
        |
        +-- MySQL: 用户、会话、消息、偏好、行程、Agent 执行记录
        |
        +-- Redis: 短期记忆、热点缓存、摘要缓存、健康状态、限流
        |
        +-- Python Agent Adapter
                |
                v
        Python AgentScope Runtime
        IntentionAgent -> OrchestrationAgent -> Skills
```

Go 后端是服务边界和数据边界，Python Runtime 是智能体能力边界。Go 不直接理解所有 Agent 内部业务，只约束输入输出协议、错误码、调用超时和执行记录。

## 后端目录规划

建议新增目录：

```text
backend/
  cmd/
    server/
      main.go
  internal/
    config/
    httpapi/
      handler/
      middleware/
      router.go
    service/
      chat.go
      memory.go
      trip.go
      preference.go
    repository/
      user_repo.go
      session_repo.go
      message_repo.go
      preference_repo.go
      trip_repo.go
      agent_run_repo.go
    cache/
      redis.go
      short_memory.go
      preference_cache.go
    adapter/
      python_agent.go
    model/
    migration/
    observability/
  migrations/
  test/
  go.mod
  go.sum
```

分层约定：

- `handler` 只处理 HTTP 入参、状态码和响应格式。
- `service` 编排业务流程，不直接写 SQL。
- `repository` 负责 MySQL 读写。
- `cache` 负责 Redis key、TTL、滑动窗口和缓存失效。
- `adapter` 负责调用 Python Agent，并隔离进程、HTTP、RPC 等具体实现。
- `model` 定义请求响应 DTO 和内部领域结构。

## 接口设计

### 通用响应

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
    "message": "Python Agent 调用超时",
    "details": {}
  }
}
```

### 健康检查

`GET /health`

返回 Go 服务、MySQL、Redis、Python Agent 的可用性。

```json
{
  "status": "ok",
  "components": {
    "mysql": "ok",
    "redis": "ok",
    "python_agent": "ok"
  }
}
```

### 聊天入口

`POST /api/v1/chat`

请求：

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

1. 校验 `user_id`、`message`，没有 `session_id` 时生成新会话。
2. 从 Redis 读取最近 10 轮短期记忆。
3. 从 MySQL 或 Redis 读取用户偏好、历史行程和摘要。
4. 调用 Python Agent Adapter。
5. 写入用户消息、助手消息、Agent 执行记录。
6. 更新 Redis 短期记忆和热点缓存。
7. 返回 Python Agent 聚合结果。

响应中的 `agent_result` 尽量兼容当前 `OrchestrationAgent` 输出：

```json
{
  "session_id": "c83a3b93",
  "message_id": "msg_xxx",
  "agent_result": {
    "status": "completed",
    "intention": {
      "intents": [],
      "key_entities": {}
    },
    "agents_executed": 2,
    "results": []
  }
}
```

### 偏好接口

- `GET /api/v1/users/:user_id/preferences`
- `PUT /api/v1/users/:user_id/preferences`
- `PATCH /api/v1/users/:user_id/preferences/:type`
- `DELETE /api/v1/users/:user_id/preferences/:type`

`PUT` 请求示例：

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

### 行程接口

- `GET /api/v1/users/:user_id/trips?limit=20&offset=0`
- `POST /api/v1/users/:user_id/trips`
- `GET /api/v1/users/:user_id/trips/:trip_id`
- `DELETE /api/v1/users/:user_id/trips/:trip_id`

### 会话与消息接口

- `POST /api/v1/sessions`
- `GET /api/v1/users/:user_id/sessions`
- `GET /api/v1/sessions/:session_id/messages?limit=50&before=msg_id`
- `DELETE /api/v1/sessions/:session_id`

### 管理与迁移接口

- `POST /api/v1/admin/migrate-memory-json`
- `GET /api/v1/admin/migrate-memory-json/status/:job_id`

管理接口默认只在开发环境启用，生产环境应通过内部网络、鉴权或一次性运维任务执行。

## MySQL 数据模型

字段类型可以在实现时按团队规范调整，以下是第一版 DDL 草案。

```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL UNIQUE,
  display_name VARCHAR(128) NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'local',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL
);

CREATE TABLE sessions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  session_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  title VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  ended_at DATETIME(6) NULL,
  INDEX idx_sessions_user_id (user_id)
);

CREATE TABLE chat_messages (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  message_id VARCHAR(128) NOT NULL UNIQUE,
  session_id VARCHAR(128) NOT NULL,
  user_id VARCHAR(128) NOT NULL,
  role VARCHAR(32) NOT NULL,
  content MEDIUMTEXT NOT NULL,
  metadata JSON NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_messages_session_created (session_id, created_at),
  INDEX idx_messages_user_created (user_id, created_at)
);

CREATE TABLE preferences (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL,
  type VARCHAR(128) NOT NULL,
  value_json JSON NOT NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'agent',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  UNIQUE KEY uk_preferences_user_type (user_id, type)
);

CREATE TABLE trip_history (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  trip_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  session_id VARCHAR(128) NULL,
  origin VARCHAR(255) NULL,
  destination VARCHAR(255) NULL,
  start_date DATE NULL,
  end_date DATE NULL,
  purpose VARCHAR(255) NULL,
  itinerary_json JSON NULL,
  raw_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX idx_trips_user_created (user_id, created_at),
  INDEX idx_trips_destination (destination)
);

CREATE TABLE agent_runs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  session_id VARCHAR(128) NULL,
  request_message_id VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL,
  request_json JSON NULL,
  intention_json JSON NULL,
  schedule_json JSON NULL,
  result_json JSON NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  duration_ms INT NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_agent_runs_session_created (session_id, created_at),
  INDEX idx_agent_runs_status_created (status, created_at)
);

CREATE TABLE user_statistics (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL UNIQUE,
  total_trips INT NOT NULL DEFAULT 0,
  total_messages INT NOT NULL DEFAULT 0,
  frequent_destinations_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL
);
```

数据映射：

- JSON 顶层 `user_id` 映射到 `users.user_id`。
- JSON `preferences[]` 映射到 `preferences`，`value` 统一写入 `value_json`。
- JSON `chat_history[]` 映射到 `chat_messages`。
- JSON `trip_history[]` 映射到 `trip_history`，原始对象保留在 `raw_json`。
- JSON `statistics` 映射到 `user_statistics`，`frequent_destinations` 写入 JSON 字段。

## Redis 设计

### Key 约定

```text
session:{session_id}:messages
user:{user_id}:preferences
user:{user_id}:summary
health:python-agent
rate_limit:user:{user_id}
agent_run:{run_id}:status
```

### 缓存策略

| Key | 类型 | TTL | 用途 |
| --- | --- | --- | --- |
| `session:{session_id}:messages` | List | 1 小时 | 最近 10 轮对话，最多 20 条消息 |
| `user:{user_id}:preferences` | String/JSON | 30 分钟 | 用户偏好热点缓存 |
| `user:{user_id}:summary` | String | 6 小时 | 长期记忆摘要 |
| `health:python-agent` | String/JSON | 30 秒 | Python Agent 健康状态 |
| `rate_limit:user:{user_id}` | String | 1 分钟 | 用户级限流计数 |
| `agent_run:{run_id}:status` | String/JSON | 24 小时 | 异步调用或排障状态 |

### 写入与失效规则

- 聊天成功后同时写 MySQL 和 Redis 短期记忆；Redis 只保留最近 20 条消息。
- 偏好更新后删除 `user:{user_id}:preferences`，下一次读取时回源 MySQL。
- 行程保存后删除 `user:{user_id}:summary`，避免摘要过期。
- Python Agent 健康检查失败时仍返回 Go 服务健康，但组件状态为 `degraded`。
- Redis 不可用时，接口应降级到 MySQL 和内存上下文，不应直接导致聊天接口不可用。

## Python Agent 适配方案

### 推荐阶段 1：Python HTTP Worker

新增一个轻量 Python 服务，例如 `python_agent_server.py`，提供：

- `GET /health`
- `POST /run`

请求：

```json
{
  "user_id": "default_user",
  "session_id": "c83a3b93",
  "message": "我想去杭州出差",
  "context": {
    "recent_messages": [],
    "preferences": {},
    "trip_history": []
  }
}
```

响应：

```json
{
  "status": "completed",
  "intention": {},
  "agents_executed": 2,
  "results": []
}
```

优点是边界清晰，Go 只需 HTTP 调用；缺点是需要为 Python Runtime 增加一个常驻进程。

### 备选阶段 1：Go 子进程调用

Go 通过 `exec.Command` 调用 Python 脚本并传入 JSON。该方式适合快速验证，但启动成本高、并发差、超时和资源控制更复杂，不建议作为长期方案。

### 阶段 2：统一 Agent SDK

把 `cli.py` 中的 `initialize_system`、`process_query` 抽成可复用模块，例如：

```text
agent_runtime.py
  class AgentRuntime:
    async def initialize(...)
    async def run(user_id, session_id, message, context) -> dict
```

CLI 和 Python HTTP Worker 共用该模块，避免逻辑分叉。

## 聊天主链路

```text
POST /api/v1/chat
  |
  |-- 1. 生成 request_id，校验参数
  |-- 2. 确保 user 和 session 存在
  |-- 3. 写入 user 消息到 MySQL
  |-- 4. 读取 Redis 最近对话，不命中则回源 MySQL
  |-- 5. 读取用户偏好和行程摘要
  |-- 6. 调用 Python Agent
  |-- 7. 写入 assistant 消息、agent_runs、trip_history/preferences 变更
  |-- 8. 刷新 Redis 短期记忆和相关缓存
  |-- 9. 返回聚合结果
```

幂等建议：

- 客户端可传 `client_message_id`，Go 后端以 `(user_id, client_message_id)` 去重。
- Python Agent 超时后，`agent_runs.status` 标记为 `timeout`，返回明确错误。
- MySQL 写入成功但 Redis 写入失败时记录日志并继续返回成功。

## JSON 迁移方案

### 迁移输入

默认扫描：

```text
data/memory/*.json
```

### 迁移规则

1. 文件名去掉 `.json` 后作为兜底 `user_id`。
2. JSON 中存在 `user_id` 时优先使用 JSON 值。
3. `preferences` 支持三种格式：
   - 新格式：`[{"type": "hotel_brands", "value": ["汉庭"]}]`
   - 旧格式：`{"hotel_brands": ["汉庭"]}`
   - 嵌套异常格式：`{"type": "preferences", "value": [...]}`
4. `chat_history` 中没有 `session_id` 时使用 `legacy_{user_id}`。
5. `trip_history.trip_id` 不存在时按顺序生成 `legacy_trip_{n}`。
6. 原始行程 JSON 保留到 `trip_history.raw_json`。
7. 迁移过程必须可重复执行，优先使用唯一键和 upsert。

### 迁移步骤

1. Dry-run：解析所有 JSON，输出用户数、消息数、偏好数、行程数、异常数。
2. Schema 初始化：执行 `backend/migrations` 下的 DDL。
3. 数据导入：按 users、sessions、preferences、chat_messages、trip_history、statistics 顺序写入。
4. 校验：对比 JSON 统计和 MySQL 聚合统计。
5. 回滚：迁移前备份数据库，迁移失败时删除本批次 `migration_batch_id` 数据。

### 验收查询

- `data/memory/default_user.json` 的 `total_messages` 与 MySQL 中对应用户消息数一致。
- JSON 中每个 `preferences.type` 在 MySQL 中都有一条记录。
- JSON 中 `trip_history.destination` 的频次能在 `user_statistics.frequent_destinations_json` 中还原。
- 重复运行迁移不产生重复消息、偏好或行程。

## 配置规划

Go 后端建议支持 `.env` 或 YAML：

```text
APP_ENV=development
HTTP_ADDR=:8080
MYSQL_DSN=user:pass@tcp(127.0.0.1:3306)/travel_agent?parseTime=true&charset=utf8mb4
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
PYTHON_AGENT_BASE_URL=http://127.0.0.1:8090
PYTHON_AGENT_TIMEOUT_SECONDS=120
CHAT_MAX_RECENT_TURNS=10
```

配置原则：

- 密钥不写入仓库。
- 本地开发提供 `.env.example`。
- 生产配置由环境变量或部署平台注入。
- 所有外部依赖都必须有超时配置。

## 可观测性与运维

### 日志

每个请求记录：

- `request_id`
- `user_id`
- `session_id`
- `path`
- `status_code`
- `duration_ms`
- `agent_run_id`
- `error_code`

### 指标

建议后续接入 Prometheus：

- HTTP 请求量、错误率、延迟。
- Python Agent 调用量、错误率、超时率、延迟。
- MySQL 查询错误数。
- Redis 命中率和错误数。
- Agent 执行状态分布：`completed`、`partial_failure`、`timeout`、`error`。

### 健康状态

- `ok`：Go、MySQL、Redis、Python Agent 均正常。
- `degraded`：Go 和 MySQL 正常，但 Redis 或 Python Agent 异常。
- `down`：Go 无法访问 MySQL，核心服务不可用。

## 安全与权限

- 所有管理接口必须默认关闭或需要管理鉴权。
- 用户数据接口后续需要接入认证，第一阶段可用开发 token 或网关鉴权占位。
- 聊天内容和 Agent 执行记录可能包含敏感信息，日志不直接打印完整 message content。
- 限流建议先按 `user_id` 和 IP 做基础限制。
- CORS 只允许明确配置的前端域名。

## 分阶段实施计划

### 阶段 0：文档与设计

产出：

- 完成本文件。
- 明确接口、表结构、缓存、迁移和验收标准。

验收：

- 团队确认 Go 后端边界不重写 Python Agent。
- 确认 MySQL/Redis/Python Adapter 三条主线可并行推进。

### 阶段 1：Go 服务骨架

任务：

- 初始化 `backend/go.mod`。
- 引入 Gin、GORM、MySQL driver、go-redis。
- 实现配置加载、结构化日志、错误响应、中间件、`GET /health`。
- 增加 Docker Compose 或本地启动说明。

验收：

- `go test ./...` 通过。
- `GET /health` 能返回 Go 服务状态。
- MySQL/Redis 不可用时返回明确组件状态。

### 阶段 2：MySQL 持久化

任务：

- 编写 migration DDL。
- 实现 users、sessions、chat_messages、preferences、trip_history、agent_runs repository。
- 实现基础 CRUD service。

验收：

- Repository 单元测试覆盖新增、查询、更新、分页、空结果。
- 偏好 `append` 和 `replace` 行为与当前 Python 记忆逻辑一致。

### 阶段 3：Redis 缓存

任务：

- 实现短期记忆 List 滑动窗口。
- 实现偏好缓存和摘要缓存。
- 实现缓存失效规则。

验收：

- 最近 10 轮对话最多保留 20 条消息。
- Redis 不可用时聊天链路可降级。
- 偏好更新后缓存会失效。

### 阶段 4：Python Agent Adapter

任务：

- 抽取 Python `AgentRuntime`。
- 新增 Python HTTP Worker 或临时子进程调用入口。
- Go 实现 adapter 调用、超时、错误映射。

验收：

- Go 调用 Python 后能获得与当前 `OrchestrationAgent` 兼容的 JSON 结果。
- Python 超时、异常、返回非法 JSON 时，Go 返回明确错误码。

### 阶段 5：聊天主接口

任务：

- 实现 `POST /api/v1/chat`。
- 串联 MySQL、Redis、Python Adapter。
- 写入 `agent_runs`。
- 更新消息和缓存。

验收：

- 使用 `default_user` 发起聊天后，MySQL 有用户消息、助手消息和 Agent 执行记录。
- 同一 session 的后续请求能读取最近上下文。
- 返回结构能兼容当前 CLI 展示所需字段。

### 阶段 6：迁移工具

任务：

- 实现 JSON 迁移命令或管理接口。
- 支持 dry-run、导入、统计校验。
- 输出迁移报告。

验收：

- `data/memory/default_user.json` 可完整导入。
- 重复导入不产生重复数据。
- 异常文件不会中断整批迁移，并能在报告中定位。

### 阶段 7：测试与文档收尾

任务：

- 补齐 API 集成测试。
- 补齐 README 后端启动说明。
- 增加接口示例和常见故障排查。

验收：

- Go 单元测试、集成测试通过。
- Python CLI 原有功能不受影响。
- 后端本地启动流程可按文档复现。

## 测试计划

### Go 单元测试

- Repository：
  - 创建用户和会话。
  - 保存并分页查询聊天消息。
  - 偏好 replace 覆盖。
  - 偏好 append 去重追加。
  - 行程保存和目的地查询。
  - Agent run 成功、失败、超时状态记录。
- Cache：
  - 短期记忆滑动窗口。
  - TTL 设置。
  - 缓存命中、回源、失效。
- Service：
  - Redis 失败降级。
  - MySQL 失败返回明确错误。
  - Python Agent 返回 partial_failure 时仍保存执行记录。

### API 集成测试

- `GET /health`
- `POST /api/v1/chat`
- `GET/PUT/PATCH/DELETE /api/v1/users/:user_id/preferences`
- `GET/POST /api/v1/users/:user_id/trips`
- `GET /api/v1/sessions/:session_id/messages`
- `POST /api/v1/admin/migrate-memory-json`

### Python 兼容测试

- 对比 CLI 与 Go 调用 Python Adapter 的输出结构。
- 验证 `results[].agent_name`、`status`、`data` 字段稳定。
- 验证偏好和行程由 Agent 输出后能被 Go 正确持久化。

### 故障测试

- MySQL 不可用。
- Redis 不可用。
- Python Agent 不可用。
- Python Agent 超时。
- Python Agent 返回非法 JSON。
- 迁移 JSON 文件格式错误。

## 错误码建议

| 错误码 | 含义 |
| --- | --- |
| `INVALID_ARGUMENT` | 请求参数无效 |
| `USER_NOT_FOUND` | 用户不存在 |
| `SESSION_NOT_FOUND` | 会话不存在 |
| `MYSQL_UNAVAILABLE` | MySQL 不可用 |
| `REDIS_UNAVAILABLE` | Redis 不可用但可能可降级 |
| `PYTHON_AGENT_UNAVAILABLE` | Python Agent 不可用 |
| `PYTHON_AGENT_TIMEOUT` | Python Agent 调用超时 |
| `PYTHON_AGENT_BAD_RESPONSE` | Python Agent 返回格式错误 |
| `MIGRATION_FAILED` | JSON 迁移失败 |
| `INTERNAL_ERROR` | 未分类内部错误 |

## 风险与应对

- Python Agent 初始化耗时较长：采用常驻 Python Worker，避免每次请求冷启动。
- LLM 调用耗时不稳定：Go Adapter 设置超时，Python 内部保留重试和熔断。
- JSON 历史数据格式不一致：迁移工具复用当前 `LongTermMemory._migrate_data` 的兼容思路，并保留原始 JSON。
- Redis 故障影响上下文：Redis 只作为缓存，必要时回源 MySQL。
- Agent 输出结构变化：用 `agent_runs.result_json` 保留原始结果，Go 只依赖最小稳定字段。
- 聊天接口写入链路较长：先保证同步正确性，后续再考虑摘要生成等异步化。

## 最终验收标准

- 本地可启动 Go 后端、MySQL、Redis、Python Agent Worker。
- `GET /health` 能反映各组件状态。
- `POST /api/v1/chat` 能调用现有 Python Agent 并返回兼容聚合结果。
- 用户消息、助手消息、Agent 执行记录能持久化到 MySQL。
- 偏好和行程能通过 API 查询，并与 Agent 更新结果一致。
- Redis 能保存最近 10 轮短期记忆，故障时可降级。
- `data/memory/default_user.json` 可迁移到 MySQL，重复迁移无重复数据。
- Go 测试和关键 Python 回归测试通过。

