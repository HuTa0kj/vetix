import re

from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path


class Base64ExecPlugin(Plugin):
    """Detect base64-decoded payloads piped directly into a shell (sh/bash)."""

    name = "Base64 Exec"

    _EXEC_RE = re.compile(
        r"base64\s+(?:-d|--decode)[^|]*?\|\s*(ba)?sh",
        re.IGNORECASE,
    )

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        issues: list[Issue] = []
        relative_path = get_relative_path(file_path, skill_dir)

        for match in self._EXEC_RE.finditer(content):
            line = content.count("\n", 0, match.start()) + 1
            issues.append(Issue(
                name="Base64 command piped to shell",
                severity=Severity.CRITICAL,
                category="Remote Execution",
                description=(
                    f"Base64-decoded command is piped into a shell ({match.group(0)})."
                ),
                file_path=relative_path,
                line=line,
                audit_required=True,
            ))
        return issues
