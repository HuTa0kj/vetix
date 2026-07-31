from langgraph.graph import StateGraph, START, END

from vetix.audit.state import SkillSafeAuditState
from vetix.audit.nodes.gather_base_info import gather_base_info
from vetix.audit.nodes.plugins_check import plugins_check
from vetix.audit.nodes.plugins_findings_verify import plugins_findings_verify
from vetix.audit.nodes.behavioral_analysis import behavioral_analysis
from vetix.audit.nodes.report import report


def skill_safe_audit_workflow():
    """SKILL security audit workflow.

    Returns:
        Compiled StateGraph
    """
    graph = StateGraph(SkillSafeAuditState)
    # nodes
    graph.add_node("gather_base_info", gather_base_info)
    graph.add_node("plugins_check", plugins_check)
    graph.add_node("plugins_findings_verify", plugins_findings_verify)
    graph.add_node("behavioral_analysis", behavioral_analysis)
    graph.add_node("report", report, defer=True)

    # edge
    graph.add_edge(START, "gather_base_info")
    graph.add_edge("gather_base_info", "plugins_check")

    graph.add_edge("plugins_check", "plugins_findings_verify")
    graph.add_edge("plugins_findings_verify", "report")

    graph.add_edge("plugins_check", "behavioral_analysis")
    graph.add_edge("behavioral_analysis", "report")
    graph.add_edge("report", END)

    return graph.compile()
