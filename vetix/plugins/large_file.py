from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path


class LargeFileCheckPlugin(Plugin):
    """Detect oversized files in SKILL directories."""

    name = "Large File"

    def __init__(self) -> None:
        self.max_bytes: int = 2 * 1024 * 1024  # 2MB

    @staticmethod
    def human_display(num_bytes: int) -> str:
        """Convert bytes into a human-readable format."""
        if num_bytes < 1024:
            return f"{num_bytes} B"
        if num_bytes < 1024 ** 2:
            return f"{num_bytes / 1024:.1f} KB"
        return f"{num_bytes / 1024 ** 2:.1f} MB"

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        size = len(content.encode("utf-8"))
        if size <= self.max_bytes:
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Large SKILL file found",
            severity=Severity.MEDIUM,
            description=f"SKILL file size {self.human_display(size)} exceeds limit of {self.human_display(self.max_bytes)}.",
            file_path=relative_path,
            audit_required=False
        )]
