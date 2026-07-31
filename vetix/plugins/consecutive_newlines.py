from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path


class ConsecutiveNewlinesCheckPlugin(Plugin):
    """Detect files with large numbers of consecutive blank lines (potential hiding of malicious content)."""

    name = "Consecutive Newlines"

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        if "\n" * 30 not in content:
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Large number of consecutive line breaks",
            severity=Severity.HIGH,
            description="The file contains a large number of consecutive newline characters, which may indicate the presence of malicious commands behind the newlines.",
            file_path=relative_path,
            audit_required=False
        )]
