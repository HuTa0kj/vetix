from typing import Annotated
from pydantic import BaseModel, Field

from langgraph.graph.message import add_messages
from langchain_core.messages import AnyMessage


class SkillSafeAuditState(BaseModel):
    model_config = {"arbitrary_types_allowed": True}

    messages: Annotated[list[AnyMessage], add_messages] = Field(default_factory=list)
    skill_dir: str
    skill_name: str = ""
    project_structure: dict = Field(default_factory=dict)
    has_scripts: bool = False
    skill_summary_report: str = ""
    skill_code_audit_report: str = ""
    task_id: str = ""
    output_dir: str = ""
    language: str = "en"
    error: str | None = None
