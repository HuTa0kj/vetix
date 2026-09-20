# Vetix — Project Map

Vetix is an LLM-agent-based scanner for [SKILL](https://docs.claude.com/en/docs/claude-code/skills) directories. It pairs deterministic plugin rules with an LLM behavioral analyst so both obvious IOCs and subtle, obfuscated attack chains get caught in one pass. Written in Go on [cloudwego/eino](https://github.com/cloudwego/eino).

End-user docs live in [README.md](./README.md) / [README_CN.md](./README_CN.md). This file is a map for navigating the codebase.

## Repository Layout

```
vetix/
  cmd/vetix/main.go     Entry point — ParseOptions → New → Run
  runner/               Options parsing, banner, version, tracing, scan orchestration
  internal/
    buildinfo/          Version / tool name constants (shared by runner and report)
    config/             YAML config loader, role → model resolution
    llm/                ChatModel factory, deep-agent construction, submit tools, middleware
    audit/              Workflow graph + the five nodes
    plugin/             Deterministic scanners + registry
    pluginutils/        Directory hash, tree with line counts, text/binary classification
    sandbox/            Read-only allow-listed filesystem.Backend for the agents
    pytext/             Python-style text for prompt rendering (must match the tuned shape exactly)
    jsonx/              Lenient JSON parsing for malformed structured output
    report/             Terminal rendering + report.json
    assets/             Embedded prompts and the helper skill
  testdata/             SKILL fixtures for plugin and pipeline tests
```

## Workflow

```
gather_base_info → plugins_check → plugins_findings_verify ┐
                                 behavioral_analysis       ┴→ report
```

State is shared via `audit.State`; every read and write goes through `compose.ProcessState`, which is the only locking eino provides. The graph is compiled with `compose.WithNodeTriggerMode(compose.AllPredecessor)` — the default `AnyPredecessor` is pregel semantics and would fire `report` before both branches finished.

Scans short-circuit on a cache hit: `runner.Run` derives the skill hash and, if `<output-dir>/<hash[:16]>/report.json` exists, renders it instead of running the pipeline. The cached report must also carry a matching `metadata.scan_key` engine fingerprint (`plugin.Fingerprint()` — build version + plugin ID set); a stale or missing fingerprint makes the cached report invalid and triggers a full re-scan that overwrites it. `-force` bypasses the cache.

- `gather_base_info` builds the tree (with per-file line counts), the directory hash, and the single-file flag.
- `plugins_check` runs every registered plugin against every file.
- `plugins_findings_verify` re-judges plugin hits against the real file content with an LLM (role `lite`). Hits with `AuditRequired=false` skip the LLM entirely.
- `behavioral_analysis` runs a deep agent over the skill to catch risks the rules miss (role `pro`). Single-file SKILLs take a fast path with no filesystem tools.
- `report` renders findings to the terminal and writes `report.json` to `<output-dir>/<directory_hash[:16]>/`. Its metadata includes `usage` — token totals captured by a per-scan model-component callback handler (`runner/usage.go`), written into `audit.State.Usage` just before rendering.

## Extension Points

- **New static rule** → add a file in `internal/plugin/` implementing `Plugin`, then register it in `internal/plugin/registry.go`. The Go binary has no runtime discovery.
- **New LLM stage** → add a function in `internal/audit/`, wire it into `internal/audit/graph.go`, and add any new fields to `audit.State`.
- **New prompt** → add a markdown file under `internal/assets/prompts/`, then load it with `assets.Prompt()`.
- **Agent tooling** → `llm.AgentOptions` controls the tool set, hidden tools, read allow-list, and iteration cap.

## Configuration

`config.yaml` is read from the working directory (override with `-config`) and defines the two LLM roles the pipeline expects:

- `roles.lite` — fast/cheap model for `plugins_findings_verify`.
- `roles.pro` — stronger reasoning model for `behavioral_analysis`.

Per-model `thinking` toggles reasoning (defaults to on for `pro`, off for `lite`), and `response_format` selects between a forced tool call and the gateway's native `json_schema` for structured output. See [example.config.yaml](./example.config.yaml).

## Design Principles

- **Defense in depth** — cheap IOCs are caught by plugins; semantic / multi-file chains by the LLM. Neither alone is sufficient.
- **Verify before reporting** — high-recall plugin hits are LLM-confirmed against real file content before reaching the report.
- **Read-only by construction** — the sandbox backend refuses every write, serves only allow-listed paths, and refuses symlinks anywhere in the path. Mutating tools are hidden from the model as a second layer. Mounting `SkillFS` does not widen access: `/skills/**` still has to be listed in `Allow`.
- **The backend is the boundary, not the model's tool list** — eino's `ToolInfos` filtering only affects visibility; a hallucinated call to a hidden tool still dispatches. Enforcement lives in `internal/sandbox`.
- **Terminal output is sanitized** — finding text comes from the scanned SKILL and is written to a terminal, so `report.Render` strips control sequences. Skipping that lets a scanned file rewrite the user's screen or clipboard.
- **Structured output with a repair net** — agents submit findings through a forced named tool; malformed arguments fall back to `internal/jsonx` so the pipeline still produces usable findings.
- **Fail closed on the review pass** — if `plugins_findings_verify` produces no parsable result, the audited hits are kept and marked unverified rather than dropped. Silently reporting nothing is the worst available outcome for a scanner.
- **Output is capped, not offloaded** — eino's large-result offload writes through the backend, which is read-only by design, so a big `read_file` would fail outright. `internal/sandbox` truncates instead and says so; the model can page with `offset`.
- **Prompt-visible text is load-bearing** — the tree representation and hit list keep their tuned shape, because the prompts were optimized against that exact text. `internal/pytext` exists for this reason.

## Guidelines

+ You should never read config.yaml. If you need to view the configuration file format, please refer to example.config.yaml.
