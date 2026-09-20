package audit

import (
	"vetix/internal/llm"
	"vetix/internal/plugin"
)

// Finding 是最终报告里的一条发现。plugin 与 behavioral 两个来源共用此结构，
// 字段即 report.json 的 findings 条目的键名。
type Finding struct {
	Source      string `json:"source"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
}

type RiskFinding = llm.RiskFinding

type BehavioralRiskItem = llm.BehavioralRiskItem

// TokenUsage 是一次扫描全部模型调用的 token 用量汇总，直接作为 report.json
// metadata.usage 的 JSON 形状。零值合法：整轮没有 LLM 调用（全部命中免复核、
// 无行为分析）就保持零值。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	// ReasoningTokens 是 completion 里思考消耗的部分，取自网关的
	// completion_token_details；不回这个字段的网关恒为零值。
	ReasoningTokens int `json:"reasoning_tokens"`
	TotalTokens     int `json:"total_tokens"`
	ModelCalls      int `json:"model_calls"`
}

type TreeStats struct {
	TopLevel int
	Files    int
	Dirs     int
}

// State 是节点间共享的可变状态。compose 只为 ProcessState / pre-state handler
// 内的访问加锁，所以所有读写都必须走这些入口，禁止直接抓指针改。
type State struct {
	TaskID     string
	SkillDir   string
	Workspace  string
	SkillName  string
	OutputDir  string
	SaveOutput bool
	DetectedAt string
	Language   string

	Tree          map[string]any
	Stats         TreeStats
	SingleSkill   bool
	SkillContent  string
	DirectoryHash string

	PluginsCheckFindings  map[string][]plugin.Issue
	PluginsVerifyFindings []RiskFinding
	LLMFindings           []*BehavioralRiskItem
	// Usage 由 report 节点在渲染前从回调采集器快照写入。report 是 AllPredecessor
	// 终端节点，此刻两条 LLM 分支都已结束，没有并发写者，直接赋值不需要锁。
	Usage TokenUsage

	Err string
}

// snapshot 是节点在锁外工作用的只读副本。Tree 与 PluginsCheckFindings 按引用共享：
// 它们写回后再无人改动，只读传出是安全的。
type snapshot struct {
	SkillDir     string
	SkillName    string
	Workspace    string
	Language     string
	SkillContent string
	Tree         map[string]any
	SingleSkill  bool
	Stats        TreeStats

	PluginsCheckFindings map[string][]plugin.Issue
}

func (s *State) snapshot() snapshot {
	return snapshot{
		SkillDir:     s.SkillDir,
		SkillName:    s.SkillName,
		Workspace:    s.Workspace,
		Language:     s.Language,
		SkillContent: s.SkillContent,
		Tree:         s.Tree,
		SingleSkill:  s.SingleSkill,
		Stats:        s.Stats,

		PluginsCheckFindings: s.PluginsCheckFindings,
	}
}

func (s *State) FileNumber() int {
	if s.SingleSkill {
		return 1
	}
	return s.Stats.Files
}

func (s snapshot) FileNumber() int {
	if s.SingleSkill {
		return 1
	}
	return s.Stats.Files
}

// Findings 按 plugin → behavioral 的顺序拼接，即报告中的排列顺序。
func (s *State) Findings() []Finding {
	out := make([]Finding, 0, len(s.PluginsVerifyFindings)+len(s.LLMFindings))
	for _, f := range s.PluginsVerifyFindings {
		out = append(out, Finding{
			Source: "plugin", Name: f.Name, Description: f.Description,
			Severity: f.Severity, Category: f.Category, FilePath: f.FilePath, Line: f.Line,
		})
	}
	for _, f := range s.LLMFindings {
		if f == nil {
			continue
		}
		out = append(out, Finding{
			Source: "behavioral", Name: f.Name, Description: f.Description,
			Severity: f.Severity, Category: f.Category, FilePath: f.FilePath, Line: f.LineNumber,
		})
	}
	return out
}
