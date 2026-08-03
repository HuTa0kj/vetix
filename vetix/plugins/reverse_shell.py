import re

from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path


class ReverseShellPlugin(Plugin):
    """Detect reverse shell patterns (bash /dev/tcp, netcat -e, socat exec:)."""

    name = "Reverse Shell"

    _SHELL_RE = re.compile(
        r"(?i)(/dev/(tcp|udp)/|\bnc\s+.*\s-e\s+|\bsocat\b.*\bexec:)"
    )

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        issues: list[Issue] = []
        relative_path = get_relative_path(file_path, skill_dir)

        for match in self._SHELL_RE.finditer(content):
            line = content.count("\n", 0, match.start()) + 1
            issues.append(Issue(
                name="Reverse shell",
                severity=Severity.CRITICAL,
                category="Remote Execution",
                description=(
                    f"Reverse shell pattern detected ({match.group(0)})."
                ),
                file_path=relative_path,
                line=line,
                audit_required=True,
            ))
        return issues
