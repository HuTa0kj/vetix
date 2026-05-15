import os.path

from langchain_core.messages import AIMessage
from deepagents import create_deep_agent, FilesystemPermission
from deepagents.backends import FilesystemBackend

from skill_scanner.skill_audit.state import SkillSafeAuditState
from skill_scanner.utils.utils import read_prompt, nodes_error, save_report, render_report, extract_report_message
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


async def skill_summary(state: SkillSafeAuditState, llm) -> dict:
    """A preliminary security analysis of SKILL was conducted.

    Args:
        state: Workflow Status
        llm: LLM

    Returns:
        Update the dictionary of status, including audit reports.
    """

    agent = _create_analyze_agent(llm, state.skill_dir)

    lang_instruction = (
        "IMPORTANT: Write the entire report in Chinese."
        if state.language == "zh"
        else "IMPORTANT: Write the entire report in English."
    )

    analyze_prompt = (
        f"{lang_instruction}\n\n"
        f"Please perform a security analysis on the following skill: \n"
        f"- SKILL Name: {state.skill_name}\n"
        f"- SKILL Table of Contents: /\n"
        f"- SKILL.md path: /SKILL.md\n"
    )

    try:
        result = await agent.ainvoke({"messages": [{"role": "user", "content": analyze_prompt}]})  # type: ignore
    except Exception as e:
        return nodes_error(f"SKILL security analysis failed: {str(e)}")

    report_text = extract_report_message(result)
    logger.info(f"SKILL security analysis complete:{state.skill_name}")
    render_report(title="SKILL Summary Report", content=report_text)
    if state.output_dir:
        save_report(state.output_dir, "skill_summary.md", report_text)
        logger.info(f"SKILL Summary Report: {os.path.join('./output', state.task_id), 'skill_summary.md'}")

    return {
        "skill_summary_report": report_text,
        "messages": [AIMessage(content=report_text)],
    }
