from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.file_classifier import is_binary_file
from vetix.utils.utils import get_relative_path


class BinaryFileCheckPlugin(Plugin):
    """Detect suspicious binary files in SKILL directories."""

    name = "Binary File"

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        if not is_binary_file(file_path):
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Binary file",
            severity=Severity.HIGH,
            category="Obfuscation",
            description="Suspicious binary files were found in the SKILL directory.",
            file_path=relative_path,
            audit_required=False
        )]
