from typing import List

from langchain.agents.middleware import ModelCallLimitMiddleware, ToolRetryMiddleware
from langchain_core.messages import HumanMessage
from deepagents import create_deep_agent, FilesystemPermission
from deepagents.backends import FilesystemBackend

from vetix.audit.state import SkillSafeAuditState
from vetix.audit.schemas import PluginsVerificationResult, RiskFinding
from vetix.middleware.tool_filter import ToolFilterMiddleware
from vetix.plugin import Issue
from vetix.llm import get_llm
from vetix.utils.logger import logger
from vetix.utils.utils import read_prompt, structured_response_repair, get_output_language


async def plugins_findings_verify(state: SkillSafeAuditState) -> dict:
    plugins_check_findings = state.plugins_check_findings
    required_verify_findings: List[Issue] = []
    plugins_verify_findings: List[RiskFinding] = []
    for _, issues in plugins_check_findings.items():
        for issue in issues:
            if issue.audit_required:
                required_verify_findings.append(issue)
            plugins_verify_findings.append(RiskFinding(
                name=issue.name,
                description=issue.description,
                severity=issue.severity,
                file_path=issue.file_path,
                line=issue.line,
            ))
    if len(required_verify_findings) == 0:
        logger.info(f"Plugin has been found to have {len(plugins_verify_findings)} security risks.")
        return {
            "plugins_verify_findings": plugins_verify_findings
        }

    workspace = state.workspace
    skill_name = state.skill_name
    project_structure = state.project_structure

    logger.info(f"Cross-validate the {len(required_verify_findings)} rules discovered by the plugin.")
    agent = create_deep_agent(
        backend=FilesystemBackend(virtual_mode=True, root_dir=workspace),
        model=get_llm(role="lite"),
        system_prompt=read_prompt("findings_verify.md"),
        permissions=[
            FilesystemPermission(
                operations=["read"],
                paths=[
                    f"/{skill_name}",
                    f"/{skill_name}**",
                ],
                mode="allow",
            ),
            FilesystemPermission(
                operations=["write"],
                paths=["/**"],
                mode="deny",
            ),
        ],
        middleware=[
            ModelCallLimitMiddleware(run_limit=50),
            ToolRetryMiddleware(
                max_retries=3,
                backoff_factor=2.0,
                initial_delay=1.0,
            ),
            ToolFilterMiddleware(
                forbidden_tools=[
                    "edit_file", "write_file", "grep", "glob",
                    "grep_search", "glob_search", "ls"
                ]
            )
        ],
        response_format=PluginsVerificationResult,
    )

    user_prompt = (
        "Please verify each of the following rules to determine if they represent a genuine risk."
        "The system combines contextual judgment with the output of the verification results for each hit, based on the schema.\n\n"
        f"SKILL absolute path: /{skill_name}\n\n"
        f"The directory structure is: {project_structure}\n\n"
        f"Hit list: \n{required_verify_findings}\n\n"
        f"{get_output_language(state.language)}\n\n"
    )

    result = await agent.ainvoke({"messages": [HumanMessage(content=user_prompt)]})  # type: ignore
    structured_response = result.get("structured_response")
    if not structured_response:
        structured_response: PluginsVerificationResult = structured_response_repair(result, PluginsVerificationResult)
        if structured_response is None:
            logger.warning(f"No structured response found for {skill_name}")
            return {
                "plugins_verify_findings": plugins_verify_findings
            }

    return {
        "plugins_verify_findings": plugins_verify_findings + structured_response.findings
    }
