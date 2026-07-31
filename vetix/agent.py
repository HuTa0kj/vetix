import os
import time
import asyncio
import uuid
from datetime import datetime
from pathlib import Path

from vetix.audit.graph import skill_safe_audit_workflow
from vetix.utils.logger import logger


def skill_analyze(
        skill_source: Path,
        workspace: str,
        language: str = "en",
        output: bool = False,
        output_dir: str = "./output",
):
    task_id = uuid.uuid4().hex
    logger.info(f"Thread ID: {task_id}")

    detected_at = datetime.now().astimezone().isoformat(timespec="seconds")

    inputs: dict = {
        "skill_dir": str(skill_source),
        "workspace": str(workspace),
        "task_id": task_id,
        "language": language,
        "save_output": output,
        "detected_at": detected_at,
    }

    if output:
        base_dir = os.path.abspath(output_dir)
        os.makedirs(base_dir, exist_ok=True)
        task_dir = os.path.join(base_dir, task_id)
        os.makedirs(task_dir, exist_ok=True)
        inputs["output_dir"] = task_dir
        logger.info(f"Output will be saved to: {task_dir}")

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
