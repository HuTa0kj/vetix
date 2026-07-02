import os

from pstruc import get_project_structure

from vetix.audit.state import SkillSafeAuditState
from vetix.utils.utils import nodes_error
from vetix.utils.logger import logger
from vetix.config import read_config


async def gather_base_info(state: SkillSafeAuditState) -> dict:
    """
    Get basic information about the SKILL catalog

    SKILL name
    Project Structure
    Is it only SKILL.md?

    Args:
        state:

    Returns:

    """
    skill_dir = state.skill_dir
    # SKILL name
    skill_name = _get_skill_name(skill_dir)

    if not skill_name:
        return nodes_error("SKILL.md not found")

    logger.info(f"Start scan SKILL: {skill_name}")

    config = read_config()
    language = config.get("language", "en")

    # Project Structure
    skill_structure = _get_skill_structure(skill_dir)

    if _is_single_skill_file(skill_dir):
        return {
            "skill_name": skill_name,
            "project_structure": skill_structure,
            "language": language,
            "single_skill_file": True,
        }

    return {
        "skill_name": skill_name,
        "project_structure": skill_structure,
        "language": language,
        "single_skill_file": False,
    }


def _get_skill_name(path: str) -> str | None:
    """
    Get the name of SKILL

    Args:
        path: SKILL path

    Returns: SKILL name

    """
    for root, dirs, files in os.walk(path):
        if "SKILL.md" in files:
            skill_name = os.path.basename(root)
            return skill_name
    return None


def _get_skill_structure(path: str) -> dict:
    """Exploration Project Structure"""
    structure: dict = get_project_structure(  # type: ignore
        start_path=path,
        output_format="dict",
        to_ignore=[
            '*.log', '*.pyc', '__pycache__', 'node_modules', '.env', 'dist', 'build', '__init__.py',
            'test', 'tests', ".git", ".github", "pyproject.toml", "LICENSE", "Dockerfile", ".DS_Store",
            "Thumbs.db", "*.pyo", "*.so", "*.dll", "*.tmp",
        ]
    )
    return structure


def _is_single_skill_file(file_path: str) -> bool:
    """Is there only one SKILL.md file in the directory?"""
    if not os.path.isdir(file_path):
        return False
    files = os.listdir(file_path)
    if len(files) > 1:
        return False
    if files[0] == "SKILL.md":
        return True
    return False
