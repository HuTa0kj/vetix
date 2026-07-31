import json
import os

from vetix.audit.nodes.gather_base_info import get_skill_file_number
from vetix.audit.schemas import RiskFinding, BehavioralRiskItem
from vetix.audit.state import SkillSafeAuditState
from vetix.plugin import Severity
from vetix.utils.logger import logger
from vetix.utils.utils import compute_directory_hash


def load_cached_report(skill_dir: str, output_dir: str) -> SkillSafeAuditState | None:
    """Load a previously saved report for `skill_dir` from `output_dir` if one exists.

    Reports are stored under <output_dir>/<directory_hash[:16]>/report.json, keyed by
    the first 16 characters of the directory content hash. Returns a reconstructed
    state ready for terminal rendering, or None when no valid cached report exists.

    Any failure (missing file, corrupt JSON, findings that no longer validate against
    the current schemas) falls back to None so the caller runs a fresh scan.
    """
    try:
        directory_hash = compute_directory_hash(skill_dir)
    except Exception:
        logger.warning(f"Failed to compute directory hash for cache lookup: {skill_dir}", exc_info=True)
        return None

    report_path = os.path.join(output_dir, directory_hash[:16], "report.json")
    if not os.path.exists(report_path):
        return None

    try:
        with open(report_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        metadata = data.get("metadata", {})
        findings = data.get("findings", []) or []

        plugin_findings: list[RiskFinding] = []
        llm_findings: list[BehavioralRiskItem] = []
        for item in findings:
            common = {
                "name": item.get("name") or "",
                "description": item.get("description") or "",
                "severity": item.get("severity") or Severity.INFO.value,
                "category": item.get("category") or "",
                "file_path": item.get("file_path") or "",
            }
            line = item.get("line") or 0
            if item.get("source") == "plugin":
                plugin_findings.append(RiskFinding(**common, line=line))
            else:
                llm_findings.append(BehavioralRiskItem(**common, line_number=line))

        file_number = metadata.get("file_number") or 0
        if not file_number:
            file_number = get_skill_file_number(skill_dir)

        return SkillSafeAuditState(
            task_id=metadata.get("task_id", ""),
            skill_dir=metadata.get("skill_dir", skill_dir),
            skill_name=metadata.get("skill_name", ""),
            file_number=file_number,
            directory_hash=metadata.get("skill_hash", directory_hash),
            output_dir=output_dir,
            save_output=False,
            detected_at=metadata.get("detected_at", ""),
            language=metadata.get("language", "en"),
            plugins_verify_findings=plugin_findings,
            llm_findings=llm_findings,
        )
    except Exception:
        logger.warning(f"Failed to load cached report: {report_path}", exc_info=True)
        return None
