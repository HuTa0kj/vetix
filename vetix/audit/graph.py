from functools import partial

from langgraph.graph import StateGraph, START, END

from vetix.model import get_llm
from vetix.audit.state import SkillSafeAuditState
from vetix.audit.nodes.get_base_info import gather_base_info
from vetix.audit.nodes.skill_summary import skill_summary
from vetix.audit.nodes.code_audit import audit_scripts
from vetix.utils.logger import logger


def _should_skip_on_error(state: SkillSafeAuditState) -> str:
    """If an error occurs in the middle of the workflow, it will end immediately."""
    if state.error:
        logger.error(state.error)
        return "end"
    return "next"


def _should_scripts_audit(state: SkillSafeAuditState) -> str:
    """If script files exist in the SKILL directory, they need to be reviewed."""
    if state.has_scripts:
        return "next"
    return "end"


def skill_safe_audit_workflow():
    """SKILL security audit workflow.

    Returns:
        Compiled StateGraph
    """
    graph = StateGraph(SkillSafeAuditState)
    # nodes
    graph.add_node("gather_base_info", gather_base_info)
    graph.add_node("skill_summary", partial(skill_summary, llm=get_llm(role="skill_summary")))
    graph.add_node("audit_scripts", partial(audit_scripts, llm=get_llm(role="audit_scripts")))

    # edge
    graph.add_edge(START, "gather_base_info")
    graph.add_conditional_edges("gather_base_info", _should_skip_on_error, {
        "next": "skill_summary",
        "end": END,
    })
    graph.add_conditional_edges("skill_summary", _should_scripts_audit, {
        "next": "audit_scripts",
        "end": END,
    })
    graph.add_edge("audit_scripts", END)

    return graph.compile()
