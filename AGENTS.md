# Vetix — Project Map

Vetix is an LLM-agent-based scanner for [SKILL](https://docs.claude.com/en/docs/claude-code/skills) directories. It pairs deterministic plugin rules with an LLM behavioral analyst so both obvious IOCs and subtle, obfuscated attack chains get caught in one pass.

End-user docs live in [README.md](./README.md) / [README_CN.md](./README_CN.md). This file is a map for navigating the codebase.

## Repository Layout

```
vetix/
  cli.py                Typer entry point — `vetix scan`
  agent.py              Orchestrator — builds the workflow, runs it
  cache.py              Loads a previously saved report.json (cache-hit fast path)
  config.py             YAML config loader (cached, sets LangSmith env)
  model.py              Role → ChatOpenAI factory (`get_llm`)
  plugin.py             Plugin ABC, Issue/Severity types, plugin loader
  audit/
    graph.py            LangGraph workflow definition
    state.py            Shared state + finding Pydantic models
    nodes/              Workflow nodes (one file per stage)
  plugins/              Deterministic scanners (auto-discovered)
  middleware/           Agent middleware (e.g. tool filtering)
  prompts/              System prompts for each LLM stage
  skills/               Helper skills mounted into the analyst agent
  utils/                Logging, file classification, helpers
config.yaml             Runtime config (gitignored) — see example.config.yaml
example.config.yaml     Config template
```

## Workflow

```
gather_base_info → plugins_check → plugins_findings_verify → behavioral_analysis → report
```

State is shared via `SkillSafeAuditState` (Pydantic). Each node returns a dict that merges into state; there are no conditional edges.

Scans short-circuit on a cache hit: before the workflow is built, `agent.py` compares `directory_hash[:16]` (computed by `gather_base_info`) against the subdirectories of the output directory and, if `<output-dir>/<hash[:16]>/report.json` exists, loads it back into state and renders it instead of running the pipeline. `--force` bypasses the cache.

- `plugins_check` runs every plugin in `vetix/plugins/` against every file.
- `plugins_findings_verify` re-judges plugin hits against the real file content with an LLM (role `lite`).
- `behavioral_analysis` runs an LLM agent (role `pro`) over the skill to catch risks the rules miss. Single-file SKILLs take a fast path with no filesystem tools.
- `report` renders findings to the terminal (via `render_report`) and writes `report.json` to `<output-dir>/<directory_hash[:16]>/`.

## Extension Points

- **New static rule** → drop a `*.py` in `vetix/plugins/`, subclass `Plugin`, implement `scan()`. Auto-discovered on the next run.
- **New LLM stage** → add a node in `vetix/audit/nodes/`, wire it into `audit/graph.py`, declare any new state fields in `audit/state.py`.
- **New prompt** → add a markdown file under `vetix/prompts/`, load it with `read_prompt()`.

## Configuration

`config.yaml` defines the two LLM roles the pipeline expects:

- `roles.lite` — fast/cheap model for `plugins_findings_verify`.
- `roles.pro` — stronger reasoning model for `behavioral_analysis`.

See [example.config.yaml](./example.config.yaml) for the full template.

## Design Principles

- **Defense in depth** — cheap IOCs are caught by plugins; semantic / multi-file chains by the LLM. Neither alone is sufficient.
- **Verify before reporting** — high-recall plugin hits are LLM-confirmed against real file content before reaching the report.
- **Read-only by construction** — every agent runs in a virtual filesystem with explicit read allow-lists, a blanket write deny, and mutating tools stripped before each model call.
- **Structured output with a repair net** — agents return Pydantic models; malformed tool calls fall back to `json_repair` so the pipeline still produces usable findings.

## Guidelines

+ You should never read config.yaml. If you need to view the configuration file format, please refer to example.config.yaml.
