from typing import Optional

from langchain_openai import ChatOpenAI

from skill_scanner.config import read_config


def get_model_info(model_id: str, config_path: Optional[str] = None) -> Optional[dict]:
    """
    Get model configuration by model id.

    Args:
        model_id: Model identifier in models list
        config_path: Optional path to config file

    Returns:
        Model configuration dictionary or None
    """
    try:
        configs = read_config(config_path=config_path)
        for model in configs.get('models', []):
            if model['id'] == model_id:
                return model
        return None
    except (KeyError, TypeError):
        return None


def get_llm(
        role: str,
        streaming: bool = False,
        config_path: Optional[str] = None,
):
    """
    Get LLM instance by role name.

    Resolves role -> model id via config roles mapping, then creates ChatOpenAI.

    Args:
        role: Role name defined in config roles section (e.g. 'main_agent')
        streaming: Enable streaming output
        config_path: Optional path to config file

    Returns:
        ChatOpenAI instance or None if configuration invalid
    """
    configs = read_config(config_path=config_path)
    roles = configs.get('roles', {})
    model_id = roles.get(role)

    if not model_id:
        print(f"No role mapping found for: {role}")
        return None

    model_info = get_model_info(model_id, config_path=config_path)
    if not model_info:
        print(f"No model info found for: {model_id} (role: {role})")
        return None

    return ChatOpenAI(
        model=model_info['id'],
        api_key=model_info['api_key'],
        base_url=model_info['base_url'],
        temperature=model_info.get('temperature', 0.7),
        streaming=streaming,
        extra_body=model_info.get('extra_body', {})
    )
