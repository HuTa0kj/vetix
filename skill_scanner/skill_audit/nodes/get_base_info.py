import os

from skill_scanner.skill_audit.state import SkillSafeAuditState
from skill_scanner.utils.utils import nodes_error
from skill_scanner.utils.logger import logger
from skill_scanner.config import read_config


async def gather_base_info(state: SkillSafeAuditState) -> dict:
    """

    Args:
        state:

    Returns:

    """
    skill_dir = state.skill_dir
    if not skill_dir or not os.path.isdir(skill_dir):
        return nodes_error(f"Invalid skill directory: {skill_dir}")

    skill_name = _get_skill_name(skill_dir)
    if not skill_name:
        return nodes_error("SKILL.md not found")

    logger.info(f"Start testing SKILL: {skill_name}")

    config = read_config()
    language = config.get("language", "en")

    return {
        "skill_name": skill_name,
        "has_scripts": _find_skill_scripts(skill_dir),
        "language": language,
    }


def _get_skill_name(path: str) -> str | None:
    """

    Args:
        path: SKILL path

    Returns: SKILL name

    """
    for root, dirs, files in os.walk(path):
        if "SKILL.md" in files:
            skill_name = os.path.basename(root)
            return skill_name
    return None


def _find_skill_scripts(path: str) -> bool:
    """
    Check if script files exist in the SKILL directory.
    Args:
        path: SKILL path

    Returns:

    """
    script_extensions = read_config()["script_extensions"]
    found_scripts = []
    for root, dirs, files in os.walk(path):
        scripts_dir = os.path.join(root, "scripts")
        if os.path.exists(scripts_dir) and os.path.isdir(scripts_dir):
            logger.info(f"Found scripts directory: {scripts_dir}")
            return True
        for file in files:
            ext = os.path.splitext(file)[1].lower()
            if ext in script_extensions:
                file_path = os.path.join(root, file)
                found_scripts.append(file_path)
        if len(found_scripts) > 0:
            logger.info(f"Found scripts file: {found_scripts[5:]}...")
            return True
    logger.info(f"No scripts found in SKILL directory: {path}")
    return False
