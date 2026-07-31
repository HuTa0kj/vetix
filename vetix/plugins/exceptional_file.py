from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.file_classifier import exist_non_text
from vetix.utils.utils import get_relative_path


class ExceptionalFileCheckPlugin(Plugin):
    """Detect files with abnormal character content (potential obfuscation attacks)."""

    name = "Exceptional File"

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        if not exist_non_text(content.encode("utf-8")):
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Exceptional file",
            severity=Severity.MEDIUM,
            category="Obfuscation",
            description="A large number of abnormal characters were found in a file that should have been readable.",
            file_path=relative_path,
            audit_required=False
        )]
