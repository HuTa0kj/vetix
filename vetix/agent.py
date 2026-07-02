import os
import time
import asyncio
import uuid
from pathlib import Path

from vetix.audit.graph import skill_safe_audit_workflow
from vetix.utils.logger import logger
from vetix.config import read_config


def skill_analyze(skill_source: Path):
    task_id = uuid.uuid4().hex
    logger.info(f"Thread ID: {task_id}")
    configs = read_config()
    language = configs.get("language", "en")
    base_output_dir = configs.get("output_dir", "./output")
    output_dir = os.path.join(base_output_dir, task_id)
    workflow = skill_safe_audit_workflow()
    start_time = time.time()
    result = asyncio.run(
        workflow.ainvoke({
            "skill_dir": str(skill_source),
            "output_dir": output_dir,
            "task_id": task_id,
        },
            config={"configurable": {"thread_id": task_id, "language": language}},
        )
    )
    end_time = time.time()
    elapsed = end_time - start_time
    logger.info(f"SKILL scan complete, time {elapsed:.2f} seconds, output directory: {output_dir}")
