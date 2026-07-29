from __future__ import annotations

import importlib
from abc import ABC, abstractmethod

from pathlib import Path
from dataclasses import dataclass
from enum import Enum


class Severity(str, Enum):
    INFO = "info"
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


@dataclass
class Issue:
    """A single finding produced by a plugin."""

    name: str
    description: str
    severity: Severity
    file_path: str
    suggestion: str
    line: int | None = 0
    audit_required: bool = True


class Plugin(ABC):
    """Base class for every plugin.

    Subclasses set ``name`` and implement :meth:`scan`. Each plugin decides
    for itself which files it applies to — typically by inspecting the
    extension via :func:`vetix.utils.utils.get_file_extension` and returning
    an empty list when the file is not relevant.
    """

    name: str = ""

    @abstractmethod
    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        """Return issues found in *content* (empty list = clean)."""
        ...


def load_plugins() -> list[Plugin]:
    """Import every module under ``vetix.plugins`` and instantiate plugins.

    Each ``*.py`` file in :mod:`vetix.plugins` (skipping ``__init__`` and
    private modules) is imported; any :class:`Plugin` subclass defined at
    module level is instantiated and collected.
    """
    instances: list[Plugin] = []
    pkg_dir = Path(__file__).resolve().parent / "plugins"
    for module_path in sorted(pkg_dir.glob("*.py")):
        stem = module_path.stem
        if stem == "__init__" or stem.startswith("_"):
            continue
        module = importlib.import_module(f"vetix.plugins.{stem}")
        for attr in vars(module).values():
            if (
                    isinstance(attr, type)
                    and issubclass(attr, Plugin)
                    and attr is not Plugin
                    and attr.__module__ == module.__name__
            ):
                instances.append(attr())
    return instances


def scan_directory(skill_dir: str) -> dict[str, list[Issue]]:
    """Walk *root* recursively, run every plugin on each file.

    Returns ``{file_path: [Issue, ...]}`` for files with at least one finding.
    Plugin exceptions are swallowed per-file so a buggy plugin can't abort the
    whole scan.
    """
    plugins = load_plugins()
    skill_dir = Path(skill_dir)
    results: dict[str, list[Issue]] = {}

    for file_path in sorted(skill_dir.rglob("*")):
        if not file_path.is_file():
            continue
        try:
            content = file_path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        plugins_issues: list[Issue] = []
        for plugin in plugins:
            try:
                plugins_issues.extend(plugin.scan(str(skill_dir), str(file_path), content))
            except Exception as exc:  # noqa: BLE001 - isolate plugin failures
                print(f"[{plugin.name}] crashed on {file_path}: {exc}")
        if plugins_issues:
            results[str(file_path)] = plugins_issues
    return results
