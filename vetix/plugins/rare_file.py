from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.file_classifier import is_risk_file
from vetix.utils.utils import get_relative_path


class RareFileCheckPlugin(Plugin):
    """SKILL documents that may contain rare auxiliary files."""

    name = "rare file"

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        if not is_risk_file(file_path):
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        return [Issue(
            name="Rare file",
            severity=Severity.MEDIUM,
            description="The SKILL directory contains rare auxiliary files that may pose a security risk.",
            file_path=relative_path,
            audit_required=False
        )]
