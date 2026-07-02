import os
from pathlib import Path
from typing import Optional

import yaml

_config_cache: dict | None = None


def read_config(config_path: Optional[str] = None) -> dict:
    """
    Read YAML configuration file (cached after first call).

    Args:
        config_path: Path to config file. If None, uses default config.yaml

    Returns:
        Configuration dictionary
    """
    global _config_cache
    if _config_cache is not None:
        return _config_cache

    if config_path:
        file_path = config_path
    else:
        file_path = Path(__file__).parent.parent / "config.yaml"

    with open(file_path, 'r', encoding='utf-8') as f:
        configs = yaml.safe_load(f)
        configs = configs or {}

    _set_langsmith_env(configs)
    _config_cache = configs
    return configs


def _set_langsmith_env(configs: dict):
    """Set LangSmith environment variables from config."""
    langsmith = configs.get('langsmith')
    if not langsmith:
        return
    if 'tracing' in langsmith:
        os.environ['LANGSMITH_TRACING'] = str(langsmith['tracing']).lower()
    if 'endpoint' in langsmith:
        os.environ['LANGSMITH_ENDPOINT'] = langsmith['endpoint']
    if 'api_key' in langsmith:
        os.environ['LANGSMITH_API_KEY'] = langsmith['api_key']
    if 'project' in langsmith:
        os.environ['LANGSMITH_PROJECT'] = langsmith['project']
