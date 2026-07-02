from pathlib import Path
from typing import Annotated

import typer

from vetix.agent import skill_analyze

app = typer.Typer(help="Skill Scan Agent — Automated scanning, identification, and assessment of SKILL security risks.")


@app.command()
def scan(
        source: Annotated[
            Path,
            typer.Option("--source", "-s", help="SKILL directory path"),
        ],
) -> None:
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

    result = skill_analyze(source)


if __name__ == "__main__":
    app()
