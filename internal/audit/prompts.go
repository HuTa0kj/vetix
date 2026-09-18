package audit

import (
	"fmt"

	"vetix/internal/plugin"
	"vetix/internal/pytext"
)

// OutputLanguage 只影响 LLM 生成的 finding 文本，不改日志、表头或 JSON 键名。
func OutputLanguage(lang string) string {
	if lang == "zh" {
		return "你需要使用中文作为输出语言。"
	}
	return "You need to use English output."
}

// VerifyPrompt 构造复核提示词。命中列表用 dataclass 风格的 repr 文本，路径前
// 硬编码一个 "/" 前缀——提示词是在这套文本上调优的，形态不能改。
func VerifyPrompt(s *State, hits []plugin.Issue) string {
	return "Please verify each of the following rules to determine if they represent a genuine risk." +
		"The system combines contextual judgment with the output of the verification results for each hit, based on the schema.\n\n" +
		fmt.Sprintf("SKILL absolute path: /%s\n\n", s.SkillName) +
		fmt.Sprintf("The directory structure is: %s\n\n", TreeRepr(s)) +
		fmt.Sprintf("Hit list: \n%s\n\n", IssueListRepr(hits)) +
		OutputLanguage(s.Language) + "\n\n"
}

// BehavioralPrompt 构造多文件行为分析的提示词。
func BehavioralPrompt(s *State) string {
	// 路径前额外拼一个 "/"：用户传绝对路径时文本会形如 //Users/...。这段畸形
	// 拼接是提示词调优时的形态，保持原样。
	return "Please perform a behavioral security analysis on the following SKILL categories to identify security risks that the rules cannot recognize.\n\n" +
		fmt.Sprintf("SKILL directory path: /%s\n\n", s.SkillDir) +
		fmt.Sprintf("The directory structure is as follows: %s\n\n", TreeRepr(s)) +
		fmt.Sprintf("The number of files in the directory is:%d\n\n", s.FileNumber()) +
		"The analysis begins with SKILL.md in the target directory.\n\n" +
		OutputLanguage(s.Language) + "\n\n"
}

// SingleFilePrompt 构造单文件快速路径的提示词，SKILL.md 全文内联。
func SingleFilePrompt(s *State) string {
	return "Please perform a behavioral security analysis on the complete content of SKILL.md below to identify security risks that the rules cannot recognize.\n\n" +
		fmt.Sprintf("SKILL directory path:/%s\n\n", s.SkillDir) +
		fmt.Sprintf("The directory structure is as follows:%s\n\n", TreeRepr(s)) +
		"The following is the full content of SKILL.md:\n\n" +
		fmt.Sprintf("```markdown\n%s\n```\n\n", s.SkillContent) +
		OutputLanguage(s.Language) + "\n\n"
}

// TreeRepr 渲染成模型看到的"目录结构"文本：单引号、多一层 "." 根、
// {'line_count': N} 叶子都要保留。
func TreeRepr(s *State) string {
	return pytext.Value(s.Tree)
}
