"""CLI 兼容入口。

当前主实现仍位于根目录 `cli.py`，这里提供标准包入口，便于后续逐步迁移。
"""

from cli import AligoCLI, main, run_health_check_standalone

__all__ = ["AligoCLI", "main", "run_health_check_standalone"]
