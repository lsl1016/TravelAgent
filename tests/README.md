# 测试目录

这里预留给后续测试代码。

建议后续拆分为：

- `tests/unit/`：纯函数、配置、解析和服务层单元测试。
- `tests/integration/`：需要 MySQL、Redis 或 Python Agent 的集成测试。
- `backend/internal/**/**_test.go`：Go 后端模块测试。

当前环境变量和外部依赖尚未统一，因此本轮只补工程结构，不执行测试。
