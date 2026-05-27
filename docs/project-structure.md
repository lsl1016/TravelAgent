# 项目结构说明

本项目当前采用“Python Agent Runtime + Go Backend”的双模块结构。

## 根目录

```text
TravelAgent/
  agents/                 # Python 核心智能体
  context/                # Python 记忆管理
  utils/                  # Python 通用工具
  travel_agent/           # Python 标准包兼容入口
  backend/                # Go HTTP 后端服务
  data/                   # 本地模型、记忆和运行数据
  docs/                   # 设计和工程文档
  scripts/                # 启动、迁移和开发辅助脚本
  tests/                  # 测试目录占位
  cli.py                  # 兼容旧入口的 Python CLI
  config.py               # 兼容旧入口的 Python 配置
  config_agentscope.py    # AgentScope 初始化配置
```

## Python 包结构化策略

当前先新增 `travel_agent/` 作为标准包入口，但不立即移动旧模块：

- `python cli.py` 继续可用。
- `python -m travel_agent` 可作为新的包入口。
- `travel_agent/config`、`travel_agent/agents`、`travel_agent/memory`、`travel_agent/utils` 先作为兼容层导出旧模块能力。

这样做的好处是可以逐步迁移 import 路径，不会一次性影响 `.claude/skills` 的动态加载。

## 后续迁移方向

后续可以逐步把旧路径迁移为：

```text
travel_agent/
  cli.py
  config/
  agents/
  memory/
  utils/
  runtime/
```

当所有引用都改为 `travel_agent.*` 后，再删除根目录的兼容入口。
