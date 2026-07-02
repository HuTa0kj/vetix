from pydantic import BaseModel, Field


class SkillDirInfo(BaseModel):
    """从用户输入中提取的 SKILL 目录"""
    has_skill: bool = Field(description="用户输入中是否包含 SKILL 相关路径或名称")
    skill_dir: str = Field(default="", description="SKILL 目录路径（如 deepforge/skills/gac-vul-review）")
