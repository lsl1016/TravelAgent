# 脚本目录

这里存放本地开发、启动和迁移辅助脚本。

## 脚本列表

- `install_python_deps.ps1`：安装 `requirements.txt` 中的 Python 依赖。
- `start_cli.ps1`：通过 `python -m travel_agent` 启动 CLI。
- `start_backend.ps1`：启动 Go 后端服务。
- `backend_build.ps1`：对 Go 后端执行 `go list ./...` 和 `go build ./...`。
- `migrate_memory_json.ps1`：调用后端管理接口迁移 `data/memory/*.json`。

## 示例

```powershell
.\scripts\install_python_deps.ps1
.\scripts\start_cli.ps1
.\scripts\start_backend.ps1 -Addr :8080
.\scripts\migrate_memory_json.ps1 -BaseUrl http://127.0.0.1:8080
.\scripts\migrate_memory_json.ps1 -BaseUrl http://127.0.0.1:8080 -Import
```
