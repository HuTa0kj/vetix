package assets

import (
	"embed"
	"fmt"
)

//go:embed prompts/*.md
var prompts embed.FS

//go:embed skills
var skills embed.FS

func Prompt(name string) (string, error) {
	b, err := prompts.ReadFile("prompts/" + name)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", name, err)
	}
	return string(b), nil
}

func Skill(name string) (string, error) {
	b, err := skills.ReadFile("skills/" + name + "/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("read skill %s: %w", name, err)
	}
	return string(b), nil
}

// SkillsFS 暴露内嵌的 skills 目录，供沙箱按 /skills/** 虚拟路径读取 helper skill。
func SkillsFS() embed.FS { return skills }
