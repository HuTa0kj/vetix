from vetix.audit.state import SkillSafeAuditState
from vetix.plugin import scan_directory
from vetix.utils.logger import logger


async def plugins_check(state: SkillSafeAuditState) -> dict:
    """Static scanning based on plugins"""
    skill_dir = state.skill_dir
    findings = scan_directory(skill_dir)
    logger.info(f"Plugin check revealed {len(findings)} security risks")
    return {"plugins_check_findings": findings}
