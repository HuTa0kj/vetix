from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path


class LongFileCheckPlugin(Plugin):
    """Detect files with excessive line counts."""

    name = "long file"

    def __init__(self) -> None:
        self.max_file_line = 3000

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        file_line = len(content.splitlines())
        if file_line <= self.max_file_line:
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Extremely long file",
            severity=Severity.MEDIUM,
            category="Obfuscation",
            description=f"An excessively long file, totaling {file_line} lines, was found in the SKILL directory.",
            file_path=relative_path,
            audit_required=False
        )]
