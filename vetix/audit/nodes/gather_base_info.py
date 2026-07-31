import os

from pstruc import get_project_structure

from vetix.audit.state import SkillSafeAuditState
from vetix.utils.utils import nodes_error, get_tree_stats, compute_directory_hash
from vetix.utils.logger import logger


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
    skill_content = _read_skill_content(skill_dir)

    # Project Structure
    skill_structure = _get_skill_structure(skill_dir)
    tree_stats = get_tree_stats(skill_structure)

    directory_hash = compute_directory_hash(skill_dir)
    logger.info(f"SKILL Hash: {directory_hash}")

    if _is_single_skill_file(skill_dir):
        logger.info("Found 1 file in the SKILL directory")
        return {
            "skill_name": skill_name,
            "project_structure": skill_structure,
            "single_skill_file": True,
            "file_number": 1,
            "skill_content": skill_content,
            "directory_hash": directory_hash,
        }
    file_number = tree_stats["total_files"]
    logger.info(f"Found {file_number} files in the SKILL directory")
    return {
        "skill_name": skill_name,
        "project_structure": skill_structure,
        "single_skill_file": False,
        "file_number": file_number,
        "skill_content": skill_content,
        "directory_hash": directory_hash,
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
    raw_structure = structure.get("structure", {})
    return _enrich_tree_with_line_counts(raw_structure, path)


def _enrich_tree_with_line_counts(structure: dict, root_path: str) -> dict:
    """The number of additional lines is added to each file node of project_structure recursively.

    Args:
        structure: `get_project_tree` returns the original directory structure (dict).
        root_path: The absolute directory path of the current layer

    Returns:
        A new dict where the leaf nodes change from None to {"line_count": N}
    """
    enriched = {}
    for key, value in structure.items():
        if isinstance(value, dict):
            sub_root = os.path.join(root_path, key)
            enriched[key] = _enrich_tree_with_line_counts(value, sub_root)
        else:
            file_full_path = os.path.join(root_path, key)
            try:
                with open(file_full_path, "r") as f:
                    line_count = sum(1 for _ in f)
            except Exception:
                line_count = 0
            enriched[key] = {"line_count": line_count}
    return enriched


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


def _read_skill_content(file_path: str) -> str:
    with open(os.path.join(file_path, "SKILL.md"), "r", encoding="utf-8") as f:
        content = f.read()
    return content
