import os.path

from langchain_core.messages import AIMessage
from deepagents import create_deep_agent, FilesystemPermission
from deepagents.backends import FilesystemBackend

from skill_scanner.utils.utils import read_prompt, nodes_error, save_report, render_report, extract_report_message
from skill_scanner.skill_audit.state import SkillSafeAuditState
from skill_scanner.utils.logger import logger


def _create_analyze_agent(llm, root_path):
    return create_deep_agent(
        model=llm,
        system_prompt=read_prompt("code_audit.md"),
        backend=FilesystemBackend(root_dir=root_path, virtual_mode=True),
        permissions=[
            FilesystemPermission(operations=["read"], paths=["/"], mode="allow"),
        ],
    )


async def audit_scripts(state: SkillSafeAuditState, llm) -> dict:
    """

    Args:
        state: Workflow Status
        llm: LLM

    Returns:

    """
    agent = _create_analyze_agent(llm, state.skill_dir)

    lang_instruction = (
        "IMPORTANT: Write the entire report in Chinese."
        if state.language == "zh"
        else "IMPORTANT: Write the entire report in English."
    )

    analyze_prompt = (
        f"{lang_instruction}\n\n"
        f"Please perform a code security analysis on the following skill: \n"
        f"- SKILL Name: {state.skill_name}\n"
        f"- SKILL Table of Contents: /\n"
        f"- SKILL.md path: /SKILL.md\n"
        f"- SKILL INFO Report: {state.skill_summary_report}\n"
    )
    try:
        result = await agent.ainvoke({"messages": [{"role": "user", "content": analyze_prompt}]})  # type: ignore
    except Exception as e:
        return nodes_error(f"SKILL code security analysis failed: {str(e)}")

    report_text = extract_report_message(result)
    logger.info(f"SKILL code security analysis complete: {state.skill_name}")
    render_report(title="SKILL Code Audit Report", content=report_text)
    if state.output_dir:
        save_report(state.output_dir, "code_audit.md", report_text)
        logger.info(f"SKILL Code Audit Report: {os.path.join('./output', state.task_id), 'code_audit.md'}")

    return {
        "skill_code_audit_report": report_text,
        "messages": [AIMessage(content=report_text)],
    }
