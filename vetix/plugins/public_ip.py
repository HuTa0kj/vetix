import re

from vetix.plugin import Issue, Plugin, Severity
from vetix.utils.utils import get_relative_path, is_public_ip


class PublicIPCheckPlugin(Plugin):
    """Check if the file contains a public IP address."""

    name = "public ip"

    def __init__(self) -> None:
        pass

    def scan(self, skill_dir: str, file_path: str, content: str) -> list[Issue]:
        issues: list[Issue] = []

        relative_path = get_relative_path(file_path, skill_dir)

        ip_pattern = r'\b(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b'
        all_ips = re.findall(ip_pattern, content)

        public_ips = []
        for ip in all_ips:
            if not is_public_ip(ip):
                continue
            public_ips.append(ip)
        if not public_ips:
            return []

        issues.append(Issue(
            name="Discover public IP address",
            severity=Severity.MEDIUM,
            description=f"Public IP addresses are often used as C2 addresses or as recipients of data breaches. {len(set(public_ips))} IP addresses were found: {set(public_ips)}",
            file_path=relative_path,
            suggestion="Even if you check suspicious IP addresses, avoid making network connections.",
            audit_required=True
        ))

        return issues
