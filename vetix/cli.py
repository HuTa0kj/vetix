from pathlib import Path
from typing import Annotated

import typer

from vetix.agent import skill_analyze
from vetix.plugin import create_plugin
from vetix.utils.banner import print_banner
from vetix.utils.logger import set_debug

app = typer.Typer(help="Vetix — Automated scanning, identification, and assessment of SKILL security risks.")


@app.command()
def scan(
        source: Annotated[
            Path,
            typer.Option("--source", "-s", help="SKILL directory path"),
        ],
        debug: Annotated[
            bool,
            typer.Option("--debug", "-d", help="Enable debug logging"),
        ] = False,
        language: Annotated[
            str,
            typer.Option("--language", "-l", help="Output language for the audit report (e.g. en, zh)"),
        ] = "en",
        output: Annotated[
            bool,
            typer.Option("--output/--no-output", "-o", help="Save the audit report to a JSON file under <output-dir>/<thread-id>/ (default: on, use --no-output to disable)"),
        ] = True,
        output_dir: Annotated[
            Path,
            typer.Option("--output-dir", "-od", help="Base directory for saved reports (default: ./output)"),
        ] = Path("./output"),
) -> None:
    if debug:
        set_debug(True)

    if not source.exists():
        typer.echo(f"Error: path not found: {source}", err=True)
        raise typer.Exit(code=1)

    if not source.is_dir():
        typer.echo(f"Error: not a directory: {source}", err=True)
        raise typer.Exit(code=1)

    skill_file = source / "SKILL.md"
    if not skill_file.exists():
        typer.echo(f"No SKILL.md found in: {source}", err=True)
        raise typer.Exit(code=1)

    print_banner()
    workspace = str(source.parent)

    result = skill_analyze(
        source,
        workspace,
        language,
        output=output,
        output_dir=str(output_dir),
    )


@app.command()
def create(
        plugin_name: Annotated[
            str,
            typer.Option("--plugin", "-p", help="Plugin name (e.g. \"my check\")"),
        ],
) -> None:
    """Create a new plugin from the template."""
    try:
        target = create_plugin(plugin_name)
        typer.echo(f"Plugin created: {target}")
    except FileExistsError as e:
        typer.echo(f"Error: {e}", err=True)
        raise typer.Exit(code=1)


if __name__ == "__main__":
    app()
