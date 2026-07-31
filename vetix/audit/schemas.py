from typing import List, Literal
from pydantic import BaseModel, Field

from vetix.plugin import Severity


class RiskFinding(BaseModel):
    name: str = Field(description="Risk Name (a short noun phrase, such as 'Reverse shell via netcat')")
    description: str = Field(
        description="Risk description (directly state the problem, do not start with a line number)")
    severity: Severity = Field(description="Severity level (low, medium, high, critical)")
    file_path: str = Field(description="File path")
    line: int = Field(description="Line number")


class PluginsVerificationResult(BaseModel):
    findings: list[RiskFinding] = Field(description="Real risk items after plugin verification")


class BehavioralAnalysisResult(BaseModel):
    """LLM output for behavioral analysis."""

    risk_found: bool = Field(description="Have any security risks been identified?")
    findings: List["BehavioralRiskItem"] = Field(
        description="List of discovered security risks"
    )


class BehavioralRiskItem(BaseModel):
    """Risk of a single action"""
    category: Literal[
        "Remote Execution",
        "Data Exfiltration",
        "Persistence",
        "Destructive",
        "Obfuscation",
        "Command Injection",
        "Privilege Escalation",
        "Sensitive File Access",
        "Network Abuse",
        "Prompt Injection",
    ] = Field(description="Risk Classification")
    severity: Literal["low", "medium", "high", "critical"] = Field(description="Severity level")
    file_path: str = Field(description="The file path where the risk is located (relative to the SKILL directory)")
    line_number: int = Field(default=0, description="Line number; enter 0 if unsure.")
    name: str = Field(description="Risk Name (a short noun phrase, such as 'Reverse shell via netcat')")
    description: str = Field(
        description="Risk description (state the problem directly, do not start with a line number)")
