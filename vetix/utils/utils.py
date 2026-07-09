import os
from pathlib import PurePath

import json_repair
from rich.console import Console
from rich.markdown import Markdown
from langchain_core.messages import AIMessage

from vetix import __version__


def get_version() -> str:
    return __version__


def get_file_extension(path: str | os.PathLike) -> str:
    """Return the lowercase file extension of *path*, including the leading dot.

    Examples:
        >>> get_file_extension("foo.py")
        '.py'
        >>> get_file_extension("README.MD")
        '.md'
        >>> get_file_extension("noext")
        ''
    """
    return PurePath(os.fspath(path)).suffix.lower()


def read_prompt(filename: str) -> str:
    """Read the prompt words from the prompt word directory.

    Args:
        filename: The filename in the prompts directory (e.g., "system.md").

    Returns:
        File content string.

    Raises:
        FileNotFoundError: If the prompt word file does not exist.
    """
    path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "prompts", filename)
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


def nodes_error(msg: str) -> dict:
    return {"error": msg, "messages": [AIMessage(content=msg)]}


def render_report(title: str, content: str):
    console = Console()
    console.rule(f"[bold cyan]{title}")
    console.print(Markdown(content))
    console.print()


def save_report(output_dir: str, filename: str, content: str):
    """Save Markdown report to file.

    Args:
        output_dir: Directory to save the report.
        filename: Filename (e.g. "skill_summary.md").
        content: Markdown content.
    """
    os.makedirs(output_dir, exist_ok=True)
    path = os.path.join(output_dir, filename)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


def extract_last_message(result: dict) -> str:
    """Extract the content of the last message from the agent results."""
    messages = result.get("messages", [])
    if messages:
        for message in reversed(messages):
            if isinstance(message, AIMessage):
                text = (message.text or "").rstrip()
                if text:
                    return text
    return ""


def extract_report_message(result: dict) -> str:
    """Extract the content of the last message from the agent results.

    `write_todos` is usually in the last message.
    If the second last AI message is longer than the last one, return the second last.
    Otherwise return the last non-empty AI message.
    """
    messages = result.get("messages", [])

    ai_messages = []
    for message in reversed(messages):
        if isinstance(message, AIMessage):
            text = (message.text or "").rstrip()
            if text:
                ai_messages.append((text, message))

    if len(ai_messages) < 2:
        return ai_messages[0][0] if ai_messages else ""

    last_text, last_msg = ai_messages[0]
    second_last_text, second_last_msg = ai_messages[1]

    if len(second_last_text) > len(last_text):
        return second_last_text
    return last_text


def get_relative_path(file_path: str, skill_dir: str) -> str:
    """Get the relative path of the file in the SKILL directory"""
    return os.path.relpath(file_path, skill_dir)


def is_public_ip(ip):
    """Determine if an IP address is a public IP address (filtering internal network and reserved addresses)."""
    try:
        parts = list(map(int, ip.split('.')))
        if len(parts) != 4:
            return False

        a, b, c, d = parts

        if not all(0 <= part <= 255 for part in parts):
            return False

        # 10.0.0.0/8
        if a == 10:
            return False
        # 172.16.0.0/12
        if a == 172 and 16 <= b <= 31:
            return False
        # 192.168.0.0/16
        if a == 192 and b == 168:
            return False

        # 127.0.0.0/8
        if a == 127:
            return False
        # 0.0.0.0/8
        if a == 0:
            return False
        # 169.254.0.0/16
        if a == 169 and b == 254:
            return False
        # 224.0.0.0/4
        if 224 <= a <= 239:
            return False
        # 240.0.0.0/4
        if 240 <= a <= 255:
            return False
        # 255.255.255.255
        if ip == "255.255.255.255":
            return False

        return True

    except (ValueError, AttributeError):
        return False


def get_skills_root() -> str:
    """Get the skills root directory."""
    return os.path.join(os.path.dirname(os.path.dirname(__file__)), "skills")



def structured_response_repair(llm_resp: dict, model_pydantic):
    """
    Some models with weaker reasoning capabilities may fail to pass Langchain structured output, resulting in data mismatches with Pydantic.
    In Langchain, structured output is essentially a tool call, allowing for manual repair using fields in the message list combined with `repair_json`.

    Args:
        llm_resp: Large model return
        model_pydantic: Data model that needs repair

    Returns:

    """
    messages = llm_resp.get("messages", [])
    model_name = getattr(model_pydantic, "__name__")
    for msg in messages:
        if isinstance(msg, AIMessage):
            invalid_tool_calls = msg.invalid_tool_calls
            for invalid_tool_call in invalid_tool_calls:
                if invalid_tool_call.get("name") == model_name:
                    json_string = invalid_tool_call.get("args")
                    repaired_dict = json_repair.repair_json(json_string, return_objects=True)
                    structured_response = model_pydantic(**repaired_dict)
                    return structured_response
    return None
