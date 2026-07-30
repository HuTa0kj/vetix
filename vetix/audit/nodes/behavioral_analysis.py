from deepagents import create_deep_agent, FilesystemPermission
from deepagents.backends import FilesystemBackend, CompositeBackend
from langchain.agents.middleware import ModelCallLimitMiddleware, ToolRetryMiddleware
from langchain_core.messages import HumanMessage, SystemMessage

from vetix.model import get_llm
from vetix.audit.state import SkillSafeAuditState, BehavioralRiskItem, BehavioralAnalysisResult
from vetix.middleware.tool_filter import ToolFilterMiddleware
from vetix.utils.logger import logger
from vetix.utils.utils import get_skills_root, read_prompt, get_tree_stats


async def behavioral_analysis(state: SkillSafeAuditState) -> dict:
    if state.single_skill_file:
        return {
            "llm_findings": await single_skill_analysis(state)
        }
    return {
        "llm_findings": await behavioral_analysis_agent(state)
    }


async def single_skill_analysis(state: SkillSafeAuditState) -> list[BehavioralRiskItem]:
    project_structure = state.project_structure
    skill_content = state.skill_content
    skill_dir = state.skill_dir
    logger.info(f"single_file_analysis: start, skill_dir={state.skill_dir}")
    if not skill_content:
        logger.warning("single_file_analysis: SKILL.md content is empty, skip")
        return []
    system_prompt = read_prompt("single_skill_analysis.md")
    user_prompt = (
        "Please perform a behavioral security analysis on the complete content of SKILL.md below to identify security risks that the rules cannot recognize.\n\n"
        f"SKILL directory path:/{skill_dir}\n\n"
        f"The directory structure is as follows:{project_structure}\n\n"
        "The following is the full content of SKILL.md:\n\n"
        f"```markdown\n{skill_content}\n```\n\n"
    )
    llm = get_llm(role="pro")
    if llm is None:
        logger.error("single_file_analysis: LLM unavailable")
        return []
    structured = llm.with_structured_output(BehavioralAnalysisResult)
    structured_response = await structured.ainvoke([
        SystemMessage(content=system_prompt),
        HumanMessage(content=user_prompt),
    ])
    findings: list[BehavioralRiskItem] = []
    if structured_response is None or not structured_response.findings:
        logger.info("single_file_analysis: no findings from LLM")
        return []
    for v in structured_response.findings:
        findings.append(BehavioralRiskItem(
            category=v.category,
            severity=v.severity,
            file_path=v.file_path or "",
            line_number=getattr(v, "line_number", 0),
            name=v.name or "",
            description=v.description or v.reasoning or "",
        ))
        logger.info(f"[LLM Behavior Analysis] {v.name}")

    return findings


async def behavioral_analysis_agent(state: SkillSafeAuditState) -> list[BehavioralRiskItem]:
    skill_dir = state.skill_dir
    workspace = state.workspace
    project_structure = state.project_structure
    file_number = state.file_number
    logger.info(f"behavioral_analysis: start, skill_dir={skill_dir}")
    user_prompt = (
        "Please perform a behavioral security analysis on the following SKILL categories to identify security risks that the rules cannot recognize.\n\n"
        f"SKILL directory path: /{skill_dir}\n\n"
        f"The directory structure is as follows: {project_structure}\n\n"
        f"The number of files in the directory is:{file_number}\n\n"
        "The analysis begins with SKILL.md in the target directory."
    )
    agent = create_deep_agent(
        backend=CompositeBackend(
            default=FilesystemBackend(virtual_mode=True, root_dir=workspace),
            routes={
                "/skills/": FilesystemBackend(root_dir=get_skills_root(), virtual_mode=True),
            },
        ),
        model=get_llm(role="pro"),
        system_prompt=read_prompt("behavioral_analysis_system.md"),
        permissions=[
            FilesystemPermission(
                operations=["read"],
                paths=[
                    f"/{skill_dir}",
                    f"/{skill_dir}/**",
                    "/skills/behavioral-analysis",
                    "/skills/behavioral-analysis/**",
                ],
                mode="allow",
            ),
            FilesystemPermission(
                operations=["write"],
                paths=["/**"],
                mode="deny",
            ),
        ],
        skills=["/skills/behavioral-analysis/"],
        middleware=[
            ToolFilterMiddleware(
                forbidden_tools=["edit_file", "write_file", "ls", "glob", "glob_search"]),
            ModelCallLimitMiddleware(run_limit=50),
            ToolRetryMiddleware(
                max_retries=3,
                backoff_factor=2.0,
                initial_delay=1.0,
            ),
        ],
        response_format=BehavioralAnalysisResult,
    )
    result = await agent.ainvoke({"messages": [HumanMessage(content=user_prompt)]})  # type: ignore
    structured_response = result.get("structured_response")
    findings: list[BehavioralRiskItem] = []
    if structured_response is None or not structured_response.findings:
        logger.info("behavioral_analysis: no findings from LLM")
        return []
    for v in structured_response.findings:
        findings.append(BehavioralRiskItem(
            category=v.category,
            severity=v.severity,
            file_path=v.file_path or "",
            line_number=getattr(v, "line_number", 0),
            name=v.name or "",
            description=v.description or v.reasoning or "",
        ))
        logger.info(f"[LLM Behavior Analysis] {v.name}")
    return findings
