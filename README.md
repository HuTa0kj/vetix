# Skill Scanner Agent

An LLM Agent-based SKILL security scanning tool for automated identification and assessment of security risks in SKILL directories.

![](./images/cover.png)

## Features

- Automatically parse SKILL directory structure and extract basic information
- Generate SKILL overview reports via LLM
- Detect script files and perform code security auditing
- Support English and Chinese report output
- LangSmith tracing integration
- Terminal report display + persistent file output

## Workflow

1. **gather_base_info** — Validate SKILL directory, extract name, detect script files
2. **skill_summary** — Perform security overview analysis via LLM Agent
3. **audit_scripts** — Perform code security auditing via LLM Agent

## Quick Start

### Prerequisites

- Python >= 3.12
- [uv](https://docs.astral.sh/uv/) (recommended package manager)

### Installation

```bash
# Clone the repository
git clone https://github.com/HuTa0kj/skill-scanner-agent.git
cd skill-scanner-agent

# Install dependencies
uv sync
```

### Configuration

Copy the example config and fill in the required fields:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to configure model API settings:

```yaml
models:
  - id: glm-5
    name: GLM-5
    api_key: ""
    base_url: ""
    temperature: 0.1

  - id: deepseek-v4-flash
    name: DeepSeek-V4-Flash
    api_key: ""
    base_url: ""
    temperature: 0.1
    extra_body: {"thinking": {"type": "disabled"}}

roles:
  skill_summary: deepseek-v4-flash
  audit_scripts: glm-5

# langsmith config (Optional)
langsmith:
  tracing: true
  endpoint: "https://api.smith.langchain.com"
  api_key: ""
  project: ""

# Script files to be detected
script_extensions: ['.py', '.sh', '.bash', '.js', '.ts', '.rb', '.pl', '.go', '.rs', '.ps1', '.cmd', '.bat']
debug: false
output_dir: "./output"
language: "en"

```

**Configuration Reference:**

| Field | Description |
|-------|-------------|
| `models` | Available LLM models, each requires `id`, `api_key`, `base_url` |
| `roles` | Role-to-model mapping, supports assigning different models for different tasks |
| `langsmith` | LangSmith tracing config (optional) |
| `script_extensions` | Script file extensions to detect |
| `output_dir` | Report output directory |
| `language` | Report language, supports `en` (English) and `zh` (Chinese) |

### Usage

```bash
# Scan a SKILL directory
skill-scanner scan --source ~/.claude/skills/skill-directory

# Or run directly
python -m skill_scanner.cli scan -s ~/.claude/skills/skill-directory
```

The target directory must contain a `SKILL.md` file.

![](./images/audit.png)

### Output

After scanning, reports are saved to `output/<task_id>/`:

```
output/
└── <task_id>/
    ├── skill_summary.md    # SKILL overview report
    └── code_audit.md       # Code security audit report (only when scripts are present)
```

## Report

![](./images/report.png)

## Tech Stack

- **LangGraph** — Workflow orchestration
- **LangChain** — LLM invocation and message management
- **DeepAgents** — Agent construction
- **Typer** — CLI framework
- **Rich** — Terminal formatted output

## License

[MIT](LICENSE)
