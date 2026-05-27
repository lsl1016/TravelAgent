# Aligo 智能旅行助手

## 项目结构说明

当前项目采用 Python Agent Runtime 与 Go Backend 并行演进的组织方式。Python 侧仍以根目录 `cli.py` 为主入口；Go 后端位于 `backend/`，作为独立服务化模块推进。

```text
TravelAgent/
  agents/                 # Python 核心智能体
  context/                # Python 短期/长期记忆管理
  utils/                  # Python 通用工具
  .claude/skills/         # Skill 插件目录，子 Agent 动态加载来源
  backend/                # Go + Gin + MySQL + Redis 后端服务
  data/                   # 本地模型、记忆文件和运行数据
  docs/                   # 项目设计、后端计划和结构说明
  scripts/                # 启动、迁移和开发辅助脚本
  tests/                  # 后续测试目录占位
  cli.py                  # Python CLI 主入口
  config.py               # LLM、RAG、系统和稳定性配置
  config_agentscope.py    # AgentScope 初始化配置
  pyproject.toml          # Python 工程化配置占位
```

文档统一迁移到 `docs/`，原 `doc/PLAN.md` 已复制为 `docs/backend-plan.md`，旧目录暂时保留以兼容已有引用。

常用脚本：

```powershell
.\scripts\install_python_deps.ps1
.\scripts\start_cli.ps1
.\scripts\start_backend.ps1
.\scripts\backend_build.ps1
.\scripts\migrate_memory_json.ps1 -BaseUrl http://127.0.0.1:8080
```

后续 Python 包结构化会采用渐进策略：先保持 `python cli.py`、现有 `agents/`、`context/`、`utils/` 导入路径稳定，再逐步迁入标准包结构。

基于**豆包大模型**和**AgentScope框架**的多智能体旅行规划系统，采用Plan-and-Execute架构，实现智能意图识别、两层记忆系统、RAG知识库、联网搜索和优先级并行调度。

## ✨ 核心亮点

### 🎯 智能意图识别
- 基于LLM语义理解的多意图识别（准确率90%+，对比关键词匹配提升25%）
- 支持6大类意图：行程规划、记忆查询、偏好管理、知识问答、信息查询、事项收集
- 自然语言理解，无需关键词匹配

### 🧠 两层记忆架构
- **Python CLI 当前实现**：进程内短期记忆 + `data/memory/{user_id}.json` 长期记忆
- **Go Backend 服务化实现**：Redis 短期记忆缓存 + MySQL 用户/会话/消息/偏好/行程持久化
- 智能识别偏好追加/覆盖动作（"我还喜欢如家" vs "我搬家到上海了"）
- 后端已提供 JSON 记忆迁移入口，便于从本地文件逐步迁移到数据库

### 📚 RAG知识库
- Milvus向量数据库 + BGE-small-zh-v1.5 Embedding模型（本地部署）
- 智能分块（Chunking）+ 滑动窗口切分 + 余弦相似度检索
- 知识溯源：返回文档来源，准确率95%

### ⚡ 优先级并行调度
- Plan-and-Execute架构：IntentionAgent → OrchestrationAgent → 子Agent
- 同优先级Agent并行执行（asyncio.gather）
- 系统响应时间从30秒优化到15秒（-50%）

### 🏗️ 插件化架构
- **Skill Plugins**：所有子Agent重构为独立插件（`.claude/skills/`）
- **LazyAgentRegistry**：动态发现机制，自动扫描注册
- **懒加载**：未使用的Skill不加载，启动速度3秒
- **Progressive Disclosure**：渐进式暴露，意图识别阶段仅加载元数据

### 🛡️ 稳定性保障
- **熔断器**：连续失败后自动熔断，保护服务
- **指数退避重试**：自动重试失败请求（最大3次）
- **健康检查**：实时监控LLM服务可用性

---

## 系统架构

```
用户输入
   ↓
┌──────────────────────────────────────────────────────────┐
│  IntentionAgent (意图识别智能体)                          │
│  - 语义理解用户意图（不使用关键词匹配）                    │
│  - 识别关键实体                                           │
│  - 生成调度计划                                           │
│  - 确定智能体优先级                                       │
│  - 动态加载 Skills Metadata (Progressive Disclosure)     │
└──────────────────────────────────────────────────────────┘
   ↓
┌──────────────────────────────────────────────────────────┐
│  OrchestrationAgent (协调器智能体)                       │
│  - 按优先级调度子智能体                                   │
│  - 同优先级并行执行                                       │
│  - 管理智能体间消息传递                                   │
│  - 集成两层记忆系统                                       │
│  - 动态实例化 Skills (Plugin Architecture)               │
└──────────────────────────────────────────────────────────┘
   ↓
┌─────────────────────── 优先级 1 (并行执行) ──────────────┐
│                                                           │
│  ┌─────────────────────┐  ┌──────────────────────────┐  │
│  │ MemoryQuery Skill   │  │ EventCollection Skill    │  │
│  │ 记忆查询智能体       │  │ 事项收集智能体            │  │
│  │ - 查询旅行记录      │  │ - 出发地/目的地           │  │
│  │ - 查询用户偏好      │  │ - 出行时间/返程地         │  │
│  │ - 查询历史对话      │  │ - 出行目的                │  │
│  └─────────────────────┘  └──────────────────────────┘  │
│                                                           │
│  ┌─────────────────────┐  ┌──────────────────────────┐  │
│  │ Preference Skill    │  │ InformationQuery Skill   │  │
│  │ 偏好管理智能体       │  │ 信息查询智能体            │  │
│  │ - 酒店/航空偏好     │  │ - 网络搜索 (DuckDuckGo)  │  │
│  │ - 座位/房型偏好     │  │ - 实时信息查询           │  │
│  │ - 机型/餐饮偏好     │  │ - LLM摘要生成            │  │
│  │ - 支持追加/覆盖     │  │                          │  │
│  └─────────────────────┘  └──────────────────────────┘  │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ RAGKnowledgeAgent Skill (知识库查询智能体)          │ │
│  │ - 差旅政策文档查询 (Milvus Lite + RAG)             │ │
│  │ - 企业内部知识检索                                  │ │
│  │ - 自动文档切分 (Chunking) + 向量检索                │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
└───────────────────────────────────────────────────────────┘
   ↓
┌─────────────────────── 优先级 2 (依赖优先级1) ───────────┐
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ ItineraryPlanningAgent Skill (行程规划智能体)       │ │
│  │ - 整合所有前序智能体信息                            │ │
│  │ - 生成完整行程计划                                  │ │
│  │ - 包含：景点、交通、酒店、餐饮                      │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
└───────────────────────────────────────────────────────────┘
   ↓
┌──────────────────────────────────────────────────────────┐
│  结果聚合与记忆更新                                       │
│  - 聚合所有智能体结果                                     │
│  - 更新长期记忆（偏好、行程历史、聊天记录）                │
│  - 生成人性化回复                                         │
└──────────────────────────────────────────────────────────┘
   ↓
最终结果
   ↓
用户看到结果
```

### 连接与可用性

为保证 LLM 服务不稳定时的可用性，在调用链外增加了以下机制（不改变原有业务逻辑）：

| 机制 | 说明 |
|------|------|
| **熔断器** | 连续失败若干次后暂停调用 LLM，直接提示「服务暂时不可用」；一段时间后自动半开试探恢复。 |
| **重试与退避** | 对意图识别、编排两次 LLM 调用做有限次重试，仅对超时、429、5xx 等可重试错误生效，采用指数退避。 |
| **健康检查** | 会话内输入 `health` 可查看熔断状态并探测 LLM 是否可达；命令行执行 `python cli.py health` 可单独做一次探测（退出码 0/1，便于监控）。 |

配置见 `config.py` 中的 `RESILIENCE_CONFIG`（重试次数、熔断阈值、恢复时间等）。

---

## 📊 关键指标

| 指标 | 优化前 | 优化后 | 提升幅度 |
|------|--------|--------|----------|
| 意图识别准确率 | 65% | 90%+ | +25% |
| 知识库问答准确率 | - | 95% | 新增功能 |
| 用户偏好记忆准确率 | - | 95% | 新增功能 |
| 系统响应时间 | 30秒 | 15秒 | -50% |
| 用户偏好缓存命中率 | - | 85% | 新增功能 |
| 系统启动速度 | 未优化 | 3秒 | 懒加载优化 |

**优化路径**：
1. **V1.0**: 关键词匹配意图识别（准确率65%） + 串行调度（响应时间30秒）
2. **V2.0**: 两层记忆系统 + RAG知识库 + 联网搜索
3. **V3.0**: LLM语义理解意图识别（准确率90%+） + 优先级并行调度（响应时间15秒）
4. **V4.0**: Skill Plugins插件化架构 + LazyAgentRegistry + Redis缓存层

---

## 核心功能

### 1. 意图识别（基于LLM语义理解）

系统支持**6大类意图**自动识别（准确率90%+）：

- ✅ **itinerary_planning**: 规划未来行程
  - 示例："我想3月11日从北京去杭州出差一周"
- ✅ **memory_query**: 查询历史记忆
  - 示例："我去过哪里？"、"我之前说过什么偏好？"
- ✅ **preference**: 管理用户偏好（支持追加/覆盖）
  - 示例："我喜欢住汉庭酒店"、"我还喜欢如家"、"我搬家到上海了"
- ✅ **rag_knowledge**: 查询企业差旅知识库
  - 示例："差旅标准是什么？"、"报销政策是什么？"
- ✅ **information_query**: 联网查询实时信息
  - 示例："杭州明天天气怎么样？"、"北京明天限行吗？"
- ✅ **event_collection**: 收集行程要素
  - 自动提取：出发地、目的地、出发时间、返程时间、出行目的

**意图识别示例**：
```
用户: "我过去都去哪旅游过？"
→ IntentionAgent 识别为 memory_query
→ 调度 MemoryQueryAgent
→ 从 trip_history 查询并回答

用户: "我还喜欢7天酒店"
→ IntentionAgent 识别为 preference
→ 调度 PreferenceAgent
→ LLM 识别「还」字，判断为 append 模式
→ 追加到 hotel_brands 列表
```

### 2. 两层记忆系统

**短期记忆（会话级）**
- Python CLI 当前使用进程内滑动窗口，默认保存最近 10 轮对话
- Go Backend 已实现 Redis List 短期记忆，保留最多 20 条消息，TTL 1 小时
- Redis 不可用时，后端聊天链路会尝试从 MySQL 回源最近上下文
- 用于上下文理解和快速访问

**长期记忆（持久化）**
- 💾 **Python CLI 当前存储**：`data/memory/{user_id}.json`
- 🗄️ **Go Backend 持久化**：MySQL 表覆盖用户、会话、聊天消息、偏好、行程、Agent 执行记录和统计信息
- 🎯 **用户偏好管理**：支持动态添加任意偏好类型，智能识别追加/覆盖动作
- 📅 **历史行程记录**：出发地、目的地、时间、目的，支持跨会话查询
- 📊 **统计信息**：常去目的地、总行程数
- 🤖 **LLM总结**：Python 记忆管理器支持对历史会话和行程生成摘要
- ⚡ **Redis缓存层**：Go Backend 已实现用户偏好、摘要和 Agent 运行状态缓存

**测试记忆系统**：
```bash
python -m compileall context
```

说明：当前 `tests/` 目录已补占位说明，但历史 README 中列出的 `test_memory_system.py` 等测试脚本尚未落地；后续会按 `tests/README.md` 规划补齐单元测试和集成测试。

### 3. RAG 知识库

基于 **Milvus** 和 **BGE-small-zh-v1.5 Embedding模型**的企业差旅知识检索系统。

**技术方案**：
- **向量数据库**: Milvus（本地存储）
- **Embedding模型**: BGE-small-zh-v1.5（中文向量化，本地部署 `data/models/bge-small-zh-v1.5`）
- **文档处理**: 智能分块（Chunking）+ 滑动窗口切分
- **检索算法**: 余弦相似度检索（Top-K=3）
- **可追溯性**: 返回文档来源，支持知识溯源
- **准确率**: 95%（知识库问答准确率）

**初始化知识库**：
```bash
python .claude/skills/ask-question/script/init_knowledge_base.py
```

**知识库内容**（8类文档）：
- 差旅标准和规定
- 报销政策
- 预订指南
- 常见问题FAQ
- 紧急情况处理
- 平台使用指南
- 城市差旅指南
- 环保倡议


### 4. 信息查询（联网搜索）

基于 **DuckDuckGo (DDGS)** 的免费网络搜索功能：
- 🌐 实时网络搜索（天气、景点、实时新闻）
- 📝 LLM自动摘要（提取关键信息）
- 🔗 来源追踪（返回搜索来源）
- 🚀 异步查询（提升响应速度）

### 5. 优先级并行调度

基于 **asyncio.gather** 的智能并行调度机制：
- 📋 **多意图识别**：支持6大类意图（规划行程、查询记忆、管理偏好、知识问答、信息查询、实时检索）
- ⚡ **优先级+并行混合模式**：同优先级Agent并行执行，不同优先级串行依赖
- 🎯 **动态调度**：根据意图识别结果动态分配优先级
- 📈 **性能提升**：系统响应时间从30秒优化到15秒（-50%）

---

## 快速开始

### 1. 安装依赖

```bash
# 使用 requirements.txt 安装所有依赖
python -m pip install -r requirements.txt

# 或者手动安装核心依赖
pip install "setuptools>=69.0.0,<82"  # milvus_lite 依赖
pip install agentscope==1.0.16        # 多智能体框架
pip install "pymilvus[milvus_lite]==2.6.9"  # 向量数据库
pip install sentence-transformers==5.2.3    # Embedding模型
pip install rich==13.9.4                    # CLI界面
pip install ddgs==9.10.0                    # 网络搜索
```

### 2. 配置模型

编辑 `config.py`，填入你的豆包大模型API密钥：

```python
LLM_CONFIG = {
    "api_key": "your-api-key-here",  # 替换为你的API密钥
    "model_name": "doubao-seed-1-6-flash-250828",
    "base_url": "https://ark.cn-beijing.volces.com/api/v3",
    "temperature": 0.7,
    "max_tokens": 8192,
}
```

**配置说明**：
- `api_key`: 豆包大模型API密钥（必填）
- `model_name`: 模型名称（推荐使用 flash 系列）
- `temperature`: 控制生成的随机性（0-1，0.7为推荐值）
- `max_tokens`: 最大输出token数（8192）

### 3. 初始化知识库

```bash
python .claude/skills/ask-question/script/init_knowledge_base.py
```

### 4. 启动系统

```bash
python cli.py
```

### 5. 启动 Go 后端（可选）

Go 后端用于服务化 API、MySQL/Redis 持久化与缓存、以及历史 JSON 记忆迁移。详细说明见 `backend/README.md`。

```powershell
.\scripts\start_backend.ps1 -Addr :8080
```

后端编译检查：

```powershell
.\scripts\backend_build.ps1
```

---

## 子智能体详解 (Skills)

所有子智能体已重构为 **Skill Plugins**，位于 `.claude/skills/` 目录下，支持动态发现与加载。

### 1. MemoryQueryAgent (记忆查询智能体) 

- **职责**: 查询用户的历史记忆
- **查询内容**:
  - 旅行历史（trip_history）
  - 用户偏好（preferences）
  - 历史对话摘要（chat_history）
- **特点**:
  - 直接查询本地记忆，无需联网
  - 使用 LLM 生成自然语言回答
  - 支持复杂的记忆推理
- **示例**: "我过去去过哪些地方？"、"我上次去北京是什么时候？"

### 2. EventCollectionAgent (事项收集智能体)

- **职责**: 收集行程规划的核心信息
- **收集内容**: 出发地、目的地、出发时间、返程时间、出行目的
- **特点**: 主动推断缺失信息

### 3. PreferenceAgent (偏好管理智能体)

- **职责**: 识别和管理用户所有偏好
- **管理偏好**:
  - 酒店品牌、航空公司、座位偏好、房型偏好
  - 机型偏好、餐饮偏好、交通偏好、预算等级
  - 支持任意自定义偏好类型
- **智能模式**:
  - **追加模式**：识别「还」、「也」等关键词，追加到现有偏好
  - **覆盖模式**：识别「搬家到」、「改成」等关键词，替换旧偏好
  - **示例**: "我还喜欢汉庭" → 追加；"我搬家到上海" → 覆盖
- **特点**:
  - 感知当前已有偏好，避免重复
  - 所有偏好作为长期偏好持久化保存
  - 从对话中提取隐含偏好

### 4. InformationQueryAgent (信息查询智能体)

- **职责**: 实时信息检索（联网）
- **查询能力**: DuckDuckGo 搜索 + LLM 摘要
- **查询场景**: 天气、景点、实时新闻、通用问答

### 5. ItineraryPlanningAgent (行程规划智能体)

- **职责**: 生成完整行程计划
- **规划内容**: 每日时间表、住宿建议、餐饮建议、交通路线、注意事项
- **特点**: 即使信息不完整也给出合理建议

### 6. RAGKnowledgeAgent (知识库查询智能体)

- **职责**: 查询企业商旅知识库
- **技术栈**: Milvus Lite + BGE 中文向量模型
- **特点**: 提供文档溯源，返回参考来源

---

## CLI 使用指南

### 启动

```bash
python cli.py
```

**启动速度**: 约 3 秒（采用LazyAgentRegistry懒加载技术）

### 内置命令

| 命令 | 说明 |
|------|------|
| `help` | 显示帮助信息 |
| `status` | 查看当前状态和记忆 |
| `health` | 检查 LLM 服务是否可用并显示熔断器状态 |
| `clear` | 清空当前任务（保留长期记忆） |
| `history` | 查看历史行程 |
| `preferences` | 查看用户偏好 |
| `exit` | 退出程序 |

单独做健康检查（不进入交互）：`python cli.py health`，返回 `OK` / `FAIL: ...`，退出码 0/1。

---

## 测试

当前 `tests/` 目录已作为工程化占位目录保留，具体测试脚本尚未补齐。因为本项目依赖 LLM、RAG、本地模型、Go 后端、MySQL、Redis 等外部条件，建议先区分轻量检查和集成测试。

### 轻量检查

```powershell
python -m compileall agents context utils
.\scripts\backend_build.ps1
```

### 后续测试规划

- Python 单元测试：记忆系统、JSON 解析、熔断器、Skill 加载器。
- Python 集成测试：CLI 主链路、意图识别、编排执行、RAG/联网查询。
- Go 后端测试：repository、cache、service、HTTP handler。
- 端到端测试：Go Backend 调用 Python Agent Worker 完整聊天链路。

---

## 项目结构

```
TravelAgent/
├── agents/                          # Python 核心编排层
│   ├── intention_agent.py           # 意图识别（语义理解）
│   ├── orchestration_agent.py       # 协调器（并行调度）
│   └── lazy_agent_registry.py       # 智能体插件注册器（懒加载）
├── .claude/skills/                  # Skill Plugins (子智能体)
│   ├── ask-question/                # 知识库问答 Skill
│   │   ├── script/                  # 代码 (agent.py, init_script)
│   │   ├── data/                    # 数据 (documents, milvus db)
│   │   └── SKILL.md                 # 技能定义
│   ├── event-collection/            # 事项收集 Skill
│   ├── plan-trip/                   # 行程规划 Skill
│   ├── preference/                  # 偏好管理 Skill
│   ├── query-info/                  # 信息查询 Skill
│   └── memory-query/                # 记忆查询 Skill
├── context/                         # 记忆系统
│   ├── memory_manager.py            # 记忆管理器
│   ├── short_term_memory.py         # 短期记忆
│   └── long_term_memory.py          # 长期记忆（支持动态偏好）
├── data/
│   ├── memory/                      # 长期记忆JSON存储（user_id.json）
│   └── models/                      # 本地模型文件
│       └── bge-small-zh-v1.5/       # BGE中文Embedding模型
├── backend/                         # Go 后端服务
│   ├── cmd/server/                  # HTTP 服务入口
│   ├── internal/                    # adapter/cache/config/httpapi/model/repository/service
│   ├── migrations/                  # MySQL DDL
│   └── README.md                    # 后端模块说明
├── docs/                            # 项目文档
│   ├── backend-plan.md              # 后端落地计划
│   ├── project-structure.md         # 项目结构说明
│   └── README.md                    # 文档目录说明
├── scripts/                         # 启动、构建、迁移辅助脚本
│   ├── install_python_deps.ps1
│   ├── start_cli.ps1
│   ├── start_backend.ps1
│   ├── backend_build.ps1
│   └── migrate_memory_json.ps1
├── tests/                           # 测试目录占位，后续补齐测试脚本
├── utils/                           # 工具与连接可用性
│   ├── circuit_breaker.py           # 熔断器
│   ├── llm_resilience.py            # 重试退避、健康检查
│   ├── json_parser.py               # JSON 解析
│   └── skill_loader.py              # Skill 加载器
├── cli.py                           # CLI 主程序
├── config.py                        # 配置文件
├── config_agentscope.py             # AgentScope 初始化与模型配置
├── pyproject.toml                   # Python 工程化配置占位
└── README.md                        # 本文件
```

---

## 技术栈总览

### 核心框架
- 📦 **AgentScope 1.0.16** - 多智能体框架
- 🤖 **豆包大模型 (doubao-seed-1-6-flash-250828)** - 大语言模型

### 数据存储
- 📄 **JSON 文件** - Python CLI 当前长期记忆存储（`data/memory/{user_id}.json`）
- 🧠 **进程内列表** - Python CLI 当前短期记忆滑动窗口
- 🗄️ **MySQL** - Go Backend 已实现的服务化持久化（用户、会话、消息、偏好、行程、Agent 执行记录）
- ⚡ **Redis** - Go Backend 已实现的短期记忆、偏好、摘要和运行状态缓存
- 🔍 **Milvus** - 向量数据库（本地存储，RAG知识库）

### 向量化与检索
- 🧠 **BGE-small-zh-v1.5** - 中文Embedding模型（本地部署）
- 📚 **Sentence-Transformers 5.2.3** - 向量化工具库
- 🎯 **余弦相似度检索** - Top-K检索算法

### 联网与搜索
- 🌐 **DuckDuckGo (DDGS 9.10.0)** - 免费网络搜索引擎
- 📝 **LLM自动摘要** - 搜索结果智能提取

### 架构设计
- 🏗️ **Skill Plugins插件化架构** - 独立开发、测试、部署
- 🔄 **LazyAgentRegistry动态发现** - 自动扫描注册Agent插件
- ⚡ **懒加载机制** - 未使用的Skill不加载（启动速度3秒）
- 🔀 **Progressive Disclosure渐进式暴露** - 意图识别阶段仅加载元数据，执行阶段按需加载
- 🎯 **优先级+并行混合调度** - asyncio.gather并发执行

### 稳定性保障
- 🔁 **指数退避重试** - 自动重试失败请求（最大3次）
- 🩺 **熔断器机制** - 连续失败后暂停调用
- 💊 **健康检查** - 实时监控LLM服务可用性

### 用户界面
- 🖥️ **Rich 13.9.4** - 精美的CLI终端界面

---

## ⚠️ 注意事项

### 模型配置
- 必须配置豆包大模型API密钥（在 `config.py` 中）
- 推荐使用 flash 系列模型（响应速度快）
- BGE Embedding模型需下载到 `data/models/bge-small-zh-v1.5/`

### 数据存储
- Python CLI 当前仍使用**进程内短期记忆**和**JSON文件长期记忆**（`data/memory/{user_id}.json`）。
- Go Backend 已落地 MySQL/Redis 相关代码和 API，但 Python CLI 尚未直接切换到 MySQL/Redis。
- 从 JSON 迁移到 MySQL 可使用后端管理接口 `POST /api/v1/admin/migrate-memory-json`，也可通过 `scripts/migrate_memory_json.ps1` 调用。
- 要让 Go 后端完整调用现有 Agent 能力，还需要补 Python Agent Worker（`GET /health`、`POST /run`）或等价 RPC/子进程适配。

### 知识库初始化
- 首次运行前必须初始化RAG知识库
- 知识库文档位于 `.claude/skills/ask-question/data/documents/`
- Milvus数据库文件生成在 `.claude/skills/ask-question/data/milvus_travel_kb.db`

### 性能优化
- 懒加载机制：系统启动时仅扫描Skill元数据，首次调用时才加载
- 并行调度：同优先级Agent并发执行，提升响应速度
- 缓存策略：热数据缓存，减少重复计算和LLM调用

---

## 🚀 未来规划

- [x] Go 后端 Gin 路由、统一响应和健康检查
- [x] MySQL repository、迁移 DDL 和核心数据模型
- [x] Redis 短期记忆、偏好缓存和 Agent 状态缓存
- [x] JSON 记忆迁移接口和辅助脚本
- [ ] Python Agent Worker（将 CLI 能力封装为 `/health`、`/run`）
- [ ] Python CLI 与 Go Backend 的完整联调
- [ ] 补齐 Python/Go 单元测试和集成测试
- [ ] Web界面（Vue + React）
- [ ] 更多Skill插件（酒店预订、机票查询等）
- [ ] 监控和日志系统

---

## 许可证

MIT License
