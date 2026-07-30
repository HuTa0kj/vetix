from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from vetix.audit.state import SkillSafeAuditState
from vetix.utils.logger import logger

_SEVERITY_STYLE = {
    "critical": "bold red",
    "high": "red",
    "medium": "yellow",
    "low": "green",
    "info": "cyan",
}


def _sev_style(severity) -> str:
    sev = severity.value if hasattr(severity, "value") else str(severity)
    return _SEVERITY_STYLE.get(sev, "white")


async def report(state: SkillSafeAuditState) -> dict:
    """Render the final audit results to the terminal."""
    console = Console()
    skill_name = state.skill_name or state.skill_dir
    plugin_findings = state.plugins_verify_findings or []
    llm_findings = state.llm_findings or []

    console.print()
    console.rule(f"[bold blue] SKILL Security Audit Report: {skill_name} [/]")
    console.print()

    # --- Basic info ---
    info_table = Table.grid(padding=(0, 2))
    info_table.add_row("[bold]Skill[/]", skill_name)
    info_table.add_row("[bold]Directory[/]", state.skill_dir)
    info_table.add_row("[bold]Files[/]", str(state.file_number))
    info_table.add_row("[bold]Language[/]", state.language or "en")
    console.print(Panel(info_table, title="[bold]Basic Info", border_style="blue"))
    console.print()

    # --- Plugin findings ---
    if plugin_findings:
        pt = Table(title="[bold]Plugin Findings", border_style="yellow", show_lines=True)
        pt.add_column("#", style="dim", width=3)
        pt.add_column("Severity", width=10)
        pt.add_column("Name", width=28, overflow="fold")
        pt.add_column("File", overflow="fold")
        pt.add_column("Line", width=6, justify="right")
        pt.add_column("Description", overflow="fold")
        pt.add_column("Suggestion", overflow="fold")
        for i, f in enumerate(plugin_findings, 1):
            sev = f.severity.value if hasattr(f.severity, "value") else str(f.severity)
            pt.add_row(
                str(i),
                f"[{_sev_style(f.severity)}]{sev}[/]",
                f.name,
                f.file_path,
                str(f.line),
                f.description,
                f.suggestion,
            )
        console.print(pt)
        console.print()
    else:
        console.print("[green]No plugin findings.[/]")
        console.print()

    # --- Behavioral findings ---
    if llm_findings:
        lt = Table(title="[bold]Behavioral Analysis Findings", border_style="magenta", show_lines=True)
        lt.add_column("#", style="dim", width=3)
        lt.add_column("Severity", width=10)
        lt.add_column("Category", width=20, overflow="fold")
        lt.add_column("Name", width=28, overflow="fold")
        lt.add_column("File", overflow="fold")
        lt.add_column("Line", width=6, justify="right")
        lt.add_column("Description", overflow="fold")
        for i, f in enumerate(llm_findings, 1):
            lt.add_row(
                str(i),
                f"[{_sev_style(f.severity)}]{f.severity}[/]",
                f.category,
                f.name,
                f.file_path,
                str(f.line_number),
                f.description,
            )
        console.print(lt)
        console.print()
    else:
        console.print("[green]No behavioral findings.[/]")
        console.print()

    console.rule("[bold blue] End of Report [/]")
    console.print()

    logger.info(
        f"Audit finished: {len(plugin_findings)} plugin findings, "
        f"{len(llm_findings)} behavioral findings"
    )
    return {}
