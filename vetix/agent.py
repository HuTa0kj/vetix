import os
import time
import asyncio
import uuid
from datetime import datetime
from pathlib import Path

from vetix.audit.graph import skill_safe_audit_workflow
from vetix.audit.nodes.report import render_report
from vetix.cache import load_cached_report
from vetix.utils.logger import logger


def skill_analyze(
        skill_source: Path,
        workspace: str,
        language: str = "en",
        output: bool = False,
        output_dir: str = "./output",
        force: bool = False,
):
    task_id = uuid.uuid4().hex
    logger.info(f"Thread ID: {task_id}")

    detected_at = datetime.now().astimezone().isoformat(timespec="seconds")

    base_dir = os.path.abspath(output_dir)
    if output:
        os.makedirs(base_dir, exist_ok=True)
        logger.info(f"Output base directory: {base_dir}")

    inputs: dict = {
        "skill_dir": str(skill_source),
        "workspace": str(workspace),
        "task_id": task_id,
        "language": language,
        "save_output": output,
        "detected_at": detected_at,
    }

    if output:
        inputs["output_dir"] = base_dir

    if not force:
        cached = load_cached_report(str(skill_source), base_dir)
        if cached is not None:
            render_report(cached)
            logger.info("Loaded cached report (use --force to re-scan)")
            return

    workflow = skill_safe_audit_workflow()
    start_time = time.time()
    result = asyncio.run(
        workflow.ainvoke(
            inputs,
            config={"configurable": {"thread_id": task_id, "workspace": workspace, "language": language}},
        )
    )
    end_time = time.time()
    elapsed = end_time - start_time
    logger.info(f"SKILL scan complete, time {elapsed:.2f} seconds")
