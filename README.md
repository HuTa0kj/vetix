# Vetix

An LLM-agent-based scanner for [SKILL](https://docs.claude.com/en/docs/claude-code/skills) directories. Vetix pairs deterministic plugin rules with an LLM behavioral analyst so that both obvious indicators of compromise and subtle, obfuscated attack chains get caught in a single pass.

[中文文档](./README_CN.md)

## Features

- **Plugin-based static scanning** — rules catch deterministic security risks.
- **LLM cross-validation** — every plugin hit is re-judged against the real file content by an LLM, so high-recall rules don't drown the final report.
- **Behavioral analysis agent** — inside a read-only virtual filesystem, traces the full chain "instruction → tool call → host impact" to uncover risks the rules miss: disguised commands, Base64 payloads, remote code loading, prompt injection, credential theft, persistence, and more.
- **Defense-in-depth sandbox** — the agent reads only inside the skill, symlinks are refused, every write is rejected at the backend, and mutating tools are hidden from the model.
- **LangSmith tracing** — every agent run is observable end-to-end.

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

Requires Go 1.25+. The module proxy must be reachable; in this environment `proxy.golang.org` is blocked, so the repo assumes `goproxy.cn`:

```bash
git clone git@github.com:HuTa0kj/vetix.git
cd vetix
go build -o vetix ./cmd/vetix
```

Prompts and the helper skill are embedded into the binary, so the executable is self-contained. Cross-compile all four targets with `./build.sh`.

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
| `langsmith` | LangSmith tracing config (optional). Note that eino's span shape differs from LangChain's, so traces are not comparable with the Python-era records. |

Common commands

```bash
# Scan a SKILL directory
./vetix -s xxx

# Enable debug logging
./vetix -s xxx -d

# Use Chinese output for the findings text
./vetix -s xxx -l zh

# Only render the report in the terminal, do not save a JSON file
./vetix -s xxx -no-output

# Custom output directory and config path
./vetix -s xxx -output-dir ./reports -c /etc/vetix/config.yaml

# Ignore a cached report and re-scan
./vetix -s xxx -force
```

Reports are written to `<output-dir>/<skill-hash-prefix>/report.json`. A second scan of an unchanged skill renders the cached report instead of re-running the pipeline; pass `-force` to bypass the cache.

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

## Adding a Plugin

Plugins live in `internal/plugin/`. The Go binary has no runtime discovery — add a file implementing `Plugin`, then register it in `internal/plugin/registry.go`:

```go
type MyCheckPlugin struct{}

func (MyCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
    // Return every hit; set AuditRequired to route a hit through LLM verification.
    return nil
}
```

## Agent Tracing

Configure [LangSmith](https://smith.langchain.com/) in `config.yaml` to trace every agent run — model calls, tool invocations, and structured outputs are all visible.

## License

[MIT](LICENSE)
