"""配置模块兼容导出。"""

from config import LLM_CONFIG, RAG_CONFIG, RESILIENCE_CONFIG, SYSTEM_CONFIG
from config_agentscope import get_model_config, init_agentscope

__all__ = [
    "LLM_CONFIG",
    "RAG_CONFIG",
    "RESILIENCE_CONFIG",
    "SYSTEM_CONFIG",
    "get_model_config",
    "init_agentscope",
]
