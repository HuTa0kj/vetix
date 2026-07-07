import logging
import os
from datetime import datetime, timezone, timedelta

from rich.console import Console
from rich.highlighter import RegexHighlighter
from rich.logging import RichHandler
from rich.theme import Theme

# UTC+8 timezone
_CST = timezone(timedelta(hours=8))

# Level tag mapping
_LEVEL_TAGS = {
    "DEBUG": "DBG",
    "INFO": "INF",
    "WARNING": "WRN",
    "ERROR": "ERR",
    "CRITICAL": "CRT",
}


class _LogHighlighter(RegexHighlighter):
    """Highlight level tags in console output."""
    highlights = [
        r"(?P<dbg>\[DBG\])",
        r"(?P<inf>\[INF\])",
        r"(?P<wrn>\[WRN\])",
        r"(?P<err>\[ERR\])",
        r"(?P<crt>\[CRT\])",
    ]


_LOG_THEME = Theme({
    "dbg": "green",
    "inf": "blue",
    "wrn": "orange1",
    "err": "red",
    "crt": "bold red",
})


class _RichFormatter(logging.Formatter):
    """Custom formatter: [TAG] [ISO timestamp] message"""

    def format(self, record: logging.LogRecord) -> str:
        tag = _LEVEL_TAGS.get(record.levelname, record.levelname)
        ts = datetime.fromtimestamp(record.created, tz=_CST).isoformat()
        return f"[{tag}] [{ts}] {record.getMessage()}"


class _LevelFilter(logging.Filter):
    """Filter out noisy third-party loggers from console."""

    def filter(self, record: logging.LogRecord) -> bool:
        return record.name not in ("httpx", "openai", "httpcore")


class Logger:
    """Rich-powered logger with file and console handlers."""

    def __init__(self, debug: bool = False, log_dir: str = "./log") -> None:
        self.debug_mode = debug
        self.logger = logging.getLogger("skill-scanner")
        self.logger.setLevel(logging.DEBUG if debug else logging.INFO)

        # Avoid duplicate handlers on repeated init
        if self.logger.handlers:
            return

        os.makedirs(log_dir, exist_ok=True)

        # Suppress noisy third-party loggers
        for name in ("httpx", "openai", "httpcore"):
            logging.getLogger(name).setLevel(logging.WARNING)

        # --- Console handler (Rich) ---
        console = Console(theme=_LOG_THEME)
        console_handler = RichHandler(
            console=console,
            show_time=False,
            show_level=False,
            show_path=False,
            markup=False,
            highlighter=_LogHighlighter(),
        )
        console_handler.setFormatter(_RichFormatter())
        console_handler.addFilter(_LevelFilter())
        console_handler.setLevel(logging.DEBUG if debug else logging.INFO)
        self.logger.addHandler(console_handler)

        # --- File handler (plain text) ---
        log_file = os.path.join(log_dir, "agent.log")
        file_handler = logging.FileHandler(log_file, mode="a", encoding="utf-8")
        file_handler.setFormatter(_RichFormatter())
        file_handler.setLevel(logging.DEBUG)
        self.logger.addHandler(file_handler)

    def get_logger(self) -> logging.Logger:
        return self.logger


# Module-level convenience instance
_default_logger = Logger()
logger = _default_logger.get_logger()


def set_debug(enabled: bool = True) -> None:
    """Enable or disable debug logging at runtime."""
    global logger, _default_logger
    _default_logger = Logger(debug=enabled)
    logger = _default_logger.get_logger()
