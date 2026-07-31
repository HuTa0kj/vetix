import ast

from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path, get_file_extension


class LowPythonCodeDensityCheckPlugin(Plugin):
    """Identify Python files with low code density"""

    name = "Low Python Code Density"

    def __init__(self) -> None:
        self.min_ratio = 0.2

    def code_density_check(self, content: str) -> bool:
        file_line = len(content.splitlines())
        if file_line < 200:
            return False
        try:
            tree = ast.parse(content)
        # A grammatical error is suspicious in itself.
        except SyntaxError:
            return True

        code_lines = set()
        for node in ast.walk(tree):
            if hasattr(node, 'lineno'):
                code_lines.add(node.lineno)

        if not code_lines:
            return True  # No valid code, highly suspicious

        max_line = max(code_lines)
        # If there is a long blank space after the "last line of code", it means there is something hidden at the end.
        trailing_blank = len(content.splitlines()) - max_line
        if trailing_blank > 30:
            return True

        # Percentage of valid lines of code to total lines of code
        ratio = len(code_lines) / len(content.splitlines())
        return ratio < self.min_ratio

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        issues: list[Issue] = []
        ext = get_file_extension(file_path)
        if ext != ".py":
            return []

        relative_path = get_relative_path(file_path, skill_dir)
        if not self.code_density_check(content):
            return []

        return [Issue(
            name="Low code density Python files",
            severity=Severity.MEDIUM,
            description="Python files often have low effective code density and may contain distracting text.",
            file_path=relative_path,
            audit_required=False
        )]
