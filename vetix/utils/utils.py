import os
from pathlib import PurePath

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
