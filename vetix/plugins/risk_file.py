from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.file_classifier import is_binary_file, is_risk_file, exist_non_text
from vetix.utils.utils import get_relative_path


class RiskFileCheckPlugin(Plugin):
    """SKILL documents that may contain risks"""

    name = "risk file"

    def __init__(self) -> None:
        self.risk_bytes: int = 2 * 1024 * 1024  # 2MB
        self.max_file_line = 3000

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        issues: list[Issue] = []
        relative_path = get_relative_path(file_path, skill_dir)
        file_line = len(content.splitlines())

        # Rare file
        if is_risk_file(file_path):
            issues.append(Issue(
                name="Rare file",
                severity=Severity.MEDIUM,
                description="The SKILL directory contains rare auxiliary files that may pose a security risk.",
                file_path=relative_path,
                suggestion="Please check the file for malicious risks and whether it is necessary.",
                audit_required=False
            ))
            return issues

        # Binary file
        if is_binary_file(file_path):
            issues.append(Issue(
                name="Binary file",
                severity=Severity.HIGH,
                description="Suspicious binary files were found in the SKILL directory.",
                file_path=relative_path,
                suggestion="Please carefully check the purpose of the binary file and verify the hash.",
                audit_required=False
            ))
            return issues

        # Large file
        size = len(content.encode("utf-8"))
        if size > self.risk_bytes:
            issues.append(Issue(
                name="Large SKILL file found",
                severity=Severity.MEDIUM,
                description=f"SKILL file size {_human_display(size)} exceeds limit of {_human_display(self.max_bytes)}.",
                file_path=relative_path,
                suggestion="Large files may contain hidden security risks; please review them carefully.",
                audit_required=False
            ))

        if file_line > self.max_file_line:
            issues.append(Issue(
                name="Extremely long file",
                severity=Severity.MEDIUM,
                description=f"An excessively long file, totaling {file_line} lines, was found in the SKILL directory.",
                file_path=relative_path,
                suggestion="Extremely long text/script files may pose security risks; please review them carefully.",
                audit_required=False
            ))
        if "\n" * 30 in content:
            issues.append(Issue(
                name="Large number of consecutive line breaks",
                severity=Severity.HIGH,
                description="The file contains a large number of consecutive newline characters, which may indicate the presence of malicious commands behind the newlines.",
                file_path=relative_path,
                suggestion="Carefully examine the file and check for malicious instructions following consecutive newline characters.",
                audit_required=False
            ))
        if exist_non_text(content.encode('utf-8')):
            issues.append(Issue(
                name="Exceptional file",
                severity=Severity.MEDIUM,
                description="A large number of abnormal characters were found in a file that should have been readable.",
                file_path=relative_path,
                suggestion="Check for unusual characters in the file to prevent obfuscation attacks.",
                audit_required=False
            ))

        return issues


def _human_display(num_bytes: int) -> str:
    """Convert bytes into a human-readable format"""
    if num_bytes < 1024:
        return f"{num_bytes} B"
    if num_bytes < 1024 ** 2:
        return f"{num_bytes / 1024:.1f} KB"
    return f"{num_bytes / 1024 ** 2:.1f} MB"
