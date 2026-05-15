# Skill Scanner Agent

Automated scanning, identification, and assessment of SKILL security risks.

## Architecture

```
skill_scanner/
  cli.py              Entry point — Typer CLI with `scan` command
  agent.py            Orchestrator — builds workflow, runs it, manages output
  skill_audit/
    graph.py          LangGraph workflow — defines node graph and conditional edges
    state.py          Shared state — SkillSafeAuditState (Pydantic model)
    nodes/
      get_base_info.py   Gather — validate skill dir, extract name, detect scripts
      skill_summary.py   Summarize — generate project overview report via LLM agent
      code_audit.py      Audit — security analysis on scripts via LLM agent
    schemas/
      info.py         Pydantic schemas for data extraction
  utils/
    utils.py          Helpers — prompt loading, report rendering/saving, message extraction
    logger.py         Rich-powered logger with console + file handlers
  prompts/
    skill_summary.md  System prompt for project analysis
    code_audit.md     System prompt for security audit
config.py             Config loader — reads config.yaml, caches result, sets LangSmith env vars
config.yaml           Runtime configuration (models, roles, langsmith, language, etc.)
```

## Workflow

```
gather_base_info -> skill_summary -> audit_scripts -> END
                   (skip if error)   (skip if no scripts)
```

## Key Design Decisions

- Each audit node creates a `deep_agent` with virtual filesystem (read-only) access to the skill directory
- Config is loaded once and cached at module level (`config.py`)
- Report language (`en`/`zh`) is controlled via `config.yaml` `language` field, injected into user prompts at runtime
- LangSmith tracing is enabled by setting env vars from config on first load
