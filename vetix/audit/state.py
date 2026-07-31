from typing import Annotated
from pydantic import BaseModel, Field

from langgraph.graph.message import add_messages
from langchain_core.messages import AnyMessage

from vetix.plugin import Issue
from vetix.audit.schemas import RiskFinding, BehavioralRiskItem


class SkillSafeAuditState(BaseModel):
    model_config = {"arbitrary_types_allowed": True}

    messages: Annotated[list[AnyMessage], add_messages] = Field(default_factory=list)

    task_id: str = ""
    skill_dir: str
    workspace: str = ""
    skill_name: str = ""
    project_structure: dict = Field(default_factory=dict)
    single_skill_file: bool = False
    file_number: int = 0
    skill_content: str = ""

    plugins_check_findings: dict[str, list[Issue]] = Field(default_factory=dict)

    plugins_verify_findings: list[RiskFinding] = Field(default_factory=list)

    llm_findings: list[BehavioralRiskItem] = Field(default_factory=list)

    output_dir: str = ""
    save_output: bool = False
    detected_at: str = ""
    language: str = "en"
    error: str | None = None
