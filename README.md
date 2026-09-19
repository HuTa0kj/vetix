# Vetix

An LLM-agent-based scanner for [SKILL](https://docs.claude.com/en/docs/claude-code/skills) directories. Vetix pairs deterministic plugin rules with an LLM behavioral analyst so that both obvious indicators of compromise and subtle, obfuscated attack chains get caught in a single pass.

[中文文档](./README_CN.md)

> [!WARNING]
> The [`examples/`](./examples) directory contains sample SKILLs for testing purposes, including deliberately crafted malicious samples. Do not install or load these SKILLs outside of a scanning test environment.

## Features

- **Plugin-based static scanning** — rules catch deterministic security risks.
- **LLM cross-validation** — every plugin hit is re-judged against the real file content by an LLM, so high-recall rules don't drown the final report.
- **Behavioral analysis agent** — inside a read-only virtual filesystem, traces the full chain "instruction → tool call → host impact" to uncover risks the rules miss: disguised commands, Base64 payloads, remote code loading, prompt injection, credential theft, persistence, and more.
- **Defense-in-depth sandbox** — the agent reads only inside the skill, symlinks are refused, every write is rejected at the backend, and mutating tools are hidden from the model.
- **LangSmith tracing** — every agent run is observable end-to-end.

## Parameter

```bash
Usage:
  vetix -s <skill-dir> | -p <preset> [flags]

Flags:
  INPUT
    -s, -source string   SKILL directory path or a skills parent directory
    -p, -preset string   Scan all skills under a preset agent skills root (claude-code, codex)

  CONFIG
    -c, -config string   Path to the YAML config file (default "./config.yaml")
    -l, -language string Output language for audit findings (en, zh) (default "en")

  OUTPUT
    -o, -output          Save the audit report to <output-dir>/<skill-hash-prefix>/report.json (default true)
    -no-output           Disable saving the audit report to a JSON file
    -output-dir string   Base directory for saved reports (default "./output")
    -force               Ignore the cached report and force a full re-scan

  DEBUG
    -d, -debug           Enable debug logging

  PLUGIN
    -pl, -plugins-list   List built-in plugins with their names and descriptions
```

## Common commands

```bash
# Scan a single SKILL; requires a SKILL.md file in the directory
vetix -s ./examples/malicious/wacli-1sk

# Scan the entire SKILL directory; requires a SKILL.md file in the immediate subdirectories
vetix -s ~/.claude/skills

# Scan using the preset path
vetix -p claude-code

# View the list of plugins
vetix -pl
```

## Detection Categories

The behavioral analysis agent classifies risks into 10 categories:

| Category | Description |
|---|---|
| Remote Execution | Remote code loading and execution, including `curl\|sh`, `wget\|bash`, and unofficial package installations |
| Data Exfiltration | Unauthorized collection and transmission of sensitive data to external addresses |
| Persistence | Backdoor mechanisms that survive reboots — crontab injection, SSH key planting, startup item modification |
| Destructive | Actions that corrupt data, delete files, or otherwise damage the host system |
| Obfuscation | Deliberate concealment of malicious payloads via Base64/Hex encoding, blank-line hiding, or disguised commands |
| Command Injection | Injection of arbitrary shell commands through unsanitized input or instruction manipulation |
| Privilege Escalation | Attempts to gain elevated permissions beyond what the skill's stated function requires |
| Sensitive File Access | Unauthorized reading or writing of SSH keys, `.aws` credentials, API keys, tokens, passwords, browser data, `.env` files, and similar secrets |
| Network Abuse | Suspicious outbound connections, C2 communication, or traffic to hard-coded external IPs/domains |
| Prompt Injection | Instructions that rewrite agent behavior — "ignore previous instructions", "DAN mode", "forget everything", etc. |

## Why an Agent?

Traditional rule-based scanners rely on predefined patterns and signatures, which limits their ability to catch novel or subtle threats. Vetix leverages LLM-powered agents to overcome these limitations:

- **Beyond rules** — Agents understand code semantics and intent, detecting malicious behaviors that rule-based approaches miss (obfuscated code, multi-step attack chains, context-aware exploits).
- **Adaptive reasoning** — Unlike static rules, agents dynamically reason about unfamiliar code patterns and adapt their analysis based on what they discover during scanning.
- **Context-aware analysis** — Agents evaluate risks in the broader context of the entire SKILL, recognizing cross-file interactions and chained vulnerabilities that individual rules cannot capture.
- **Natural-language explanations** — Every finding comes with a clear, human-readable explanation of the risk, impact, and recommended remediation — not just a rule ID.

## Deployment

### Build

Requires Go 1.25+:

```bash
git clone git@github.com:HuTa0kj/vetix.git
cd vetix
go build -o vetix ./cmd/vetix
```

Copy the example config and fill in your model credentials:

```bash
cp example.config.yaml config.yaml
```

`config.yaml` is read from the current working directory by default (override with `-config`). It defines two LLM roles: a lightweight model for plugin-hit verification, and a stronger model for behavioral analysis.

```yaml
models:
  - id: deepseek-v4-pro
    name: DeepSeek-V4-Pro
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {}
    extra_headers: {}       # appended to every request (fast path and agent calls alike)
    thinking: true          # inject {"thinking": {"type": "enabled"}} into the request body

  - id: deepseek-v4-flash
    name: DeepSeek-V4-Flash
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {}
    thinking: false

roles:
  lite: deepseek-v4-flash
  pro:  deepseek-v4-pro

# Optional: LangSmith tracing
langsmith:
  tracing: false
  endpoint: "https://api.smith.langchain.com"
  api_key: ""
  project: ""
```

| Field | Description |
|-------|-------------|
| `models` | Available LLMs. Each entry requires `id`, `api_key`, `base_url`; `temperature`, `extra_body`, `extra_headers`, `thinking` and `response_format` are optional. `base_url` must point at the API root — most gateways need the `/v1` suffix, otherwise they serve a web page instead of JSON. |
| `models[].extra_headers` | Extra HTTP headers sent with every request, e.g. gateway routing keys. Applied at the transport layer, so agent-internal model calls get them too. |
| `models[].thinking` | Whether to request reasoning. Defaults to on for the `pro` role and off for `lite`. Note that some gateways reject a forced tool call while reasoning is enabled; the single-file fast path detects this and falls back to `response_format` automatically. |
| `models[].response_format` | `tool` (default) forces a named tool call for structured output; `json_schema` uses the gateway's native `response_format`. |
| `roles.lite` | Fast model, for plugin-hit verification. |
| `roles.pro` | Reasoning model, for behavioral analysis. |
| `langsmith` | LangSmith tracing config (optional) |

### Docker

Build the image

```bash
docker build -t vetix:latest .
```

Configuration

```bash
cp example.config.yaml config.yaml
# edit config.yaml: fill in real api_key / base_url for both models
```

Run a scan

```bash
docker run --rm \
  -v "$PWD/config.yaml:/work/config.yaml:ro" \
  -v "$PWD/examples/skills/xxx:/skills/xxx:ro" \
  -v "$PWD/output:/work/output" \
  vetix:latest -s /skills/xxx
```

### Docker Compose

```bash
docker compose run --rm vetix -s /skills/xxx
```

## Built-in Plugins

Nine deterministic rules are compiled into the binary. Every plugin runs against every file in the SKILL directory; hits marked **LLM-verified** are re-judged against the real file content by the verification pass before reaching the report, the rest go straight into it.

| Plugin | Detects | Default severity | LLM-verified |
|---|---|---|---|
| `base64_exec` | A Base64-decoded command piped into a shell | critical | yes |
| `reverse_shell` | Reverse-shell patterns — `/dev/tcp`, `nc -e`, `socat exec:` | critical | yes |
| `binary_file` | Binary files inside the SKILL directory | high | no |
| `consecutive_newlines` | Runs of 30+ consecutive newlines hiding content | high | no |
| `exceptional_file` | Text files full of non-printable characters | medium | no |
| `large_file` | Single files over 2 MB | medium | no |
| `long_file` | Single files over 3000 lines | medium | no |
| `public_ip` | Hard-coded public IPv4 addresses | medium | yes |
| `rare_file` | Files whose extension is outside the text whitelist | medium | no |

## Adding a Plugin

Plugins are located in `internal/plugin/`. When adding a new file that implements `Plugin`, you need to register it in `internal/plugin/registry.go`:

```go
package plugin

type MyCheckPlugin struct{}

func (MyCheckPlugin) Meta() Meta {
    return Meta{
        ID:          "my_check",
        Name:        "My Check",
        Description: "One sentence on what this rule detects.",
    }
}

func (MyCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
    // Return every hit; set AuditRequired to route a hit through LLM verification.
    return nil
}
```

## License

[MIT](LICENSE)
