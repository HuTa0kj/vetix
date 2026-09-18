package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/projectdiscovery/gologger"

	"vetix/internal/assets"
	"vetix/internal/llm"
	"vetix/internal/plugin"
)

// maxModelCalls 是每轮的模型调用上限。注意提示词里写的是"50 tool calls"，
// 而这里限制的是模型调用次数，两者本来就不一致；按代码的语义实现。
const maxModelCalls = 50

// VerifyFindings 复核插件命中：无需复核的直接进入结果，其余交给 lite 角色判定。
func VerifyFindings(ctx context.Context, h *stateHandle, f *llm.Factory) error {
	snap, err := h.snapshot()
	if err != nil {
		return err
	}

	var direct []RiskFinding
	var needVerify []plugin.Issue
	for _, issues := range snap.PluginsCheckFindings {
		for _, i := range issues {
			if i.AuditRequired {
				needVerify = append(needVerify, i)
				continue
			}
			direct = append(direct, RiskFinding{
				Name: i.Name, Description: i.Description, Severity: string(i.Severity),
				Category: i.Category, FilePath: i.FilePath, Line: i.Line,
			})
		}
	}
	if len(needVerify) == 0 {
		gologger.Info().Msgf("Plugin has been found to have %d security risks", len(direct))
		return h.with(func(s *State) { s.PluginsVerifyFindings = direct })
	}
	gologger.Info().Msgf("Cross-validate the %d rules discovered by the plugin", len(needVerify))

	instruction, err := assets.Prompt("findings_verify.md")
	if err != nil {
		return err
	}
	submit, collector, err := llm.NewSubmitTool(llm.SubmitVerifyTool,
		"Submit the verification result with every finding you confirmed.")
	if err != nil {
		return err
	}
	agent, _, err := f.NewAgent(ctx, "lite", llm.AgentOptions{
		Name:           "findings_verify",
		Instruction:    instruction + fmt.Sprintf(llm.SubmitInstruction, llm.SubmitVerifyTool),
		Tools:          []tool.BaseTool{submit},
		ReturnDirectly: []string{llm.SubmitVerifyTool},
		// 复核阶段只保留 read_file。
		HiddenTools: []string{"edit_file", "write_file", "grep", "glob", "ls"},
		BackendRoot: snap.Workspace,
		Allow:       []string{"/" + snap.SkillName},
		MaxIters:    maxModelCalls,
	})
	if err != nil {
		return err
	}

	if _, runErr := RunAgent(ctx, agent, VerifyPrompt(snap, needVerify)); runErr != nil && !isSoftStop(runErr) {
		return runErr
	}
	verified, ok := ParseVerifyFindings(collector.Raw())
	if !ok {
		// 复核结果没拿到：可能模型真的没确认任何命中，也可能结构化输出没解析出来，
		// 而后者会让需要复核的命中整批消失——对安全扫描器来说是 fail-open。这里退回
		// 保留原始命中并标注未复核：宁可多报几条待人工判断的，也不能因为一次畸形输出
		// 就把 critical 命中清零。
		gologger.Warning().Msgf("No structured verification result for %s; keeping %d audited hits as unverified",
			snap.SkillName, len(needVerify))
		verified = unverifiedHits(needVerify)
	}
	return h.with(func(s *State) { s.PluginsVerifyFindings = append(direct, verified...) })
}

// unverifiedHits 把未经复核的原始命中转成保留项，并在描述里写明它没经过 LLM 确认，
// 免得报告读者把降级结果当成复核通过。
func unverifiedHits(issues []plugin.Issue) []RiskFinding {
	out := make([]RiskFinding, 0, len(issues))
	for _, i := range issues {
		out = append(out, RiskFinding{
			Name:        i.Name,
			Description: i.Description + " (unverified: the review pass produced no parsable result)",
			Severity:    string(i.Severity),
			Category:    i.Category,
			FilePath:    i.FilePath,
			Line:        i.Line,
		})
	}
	return out
}

// BehavioralAnalysis：单文件走无工具的快速路径，多文件走带沙箱的 agent。
func BehavioralAnalysis(ctx context.Context, h *stateHandle, f *llm.Factory) error {
	snap, err := h.snapshot()
	if err != nil {
		return err
	}

	var findings []*BehavioralRiskItem
	if snap.SingleSkill {
		findings, err = singleFileAnalysis(ctx, snap, f)
	} else {
		findings, err = behavioralAgent(ctx, snap, f)
	}
	if err != nil {
		return err
	}
	return h.with(func(s *State) { s.LLMFindings = findings })
}

func singleFileAnalysis(ctx context.Context, s snapshot, f *llm.Factory) ([]*BehavioralRiskItem, error) {
	if s.SkillContent == "" {
		gologger.Warning().Msg("single_file_analysis: SKILL.md content is empty, skip")
		return nil, nil
	}
	gologger.Info().Msgf("single_file_analysis: start, skill_dir=%s", s.SkillDir)
	instruction, err := assets.Prompt("single_skill_analysis.md")
	if err != nil {
		return nil, err
	}
	cm, m, err := f.Role(ctx, "pro")
	if err != nil {
		return nil, err
	}

	// 依次尝试各档结构化输出方案。网关对"思考模式 + 强制工具调用"的支持差别很大，
	// 被拒时降级而不是整轮失败。
	findings, err := runAttempts(ctx, fastPathAttempts(cm, m, instruction, SingleFilePrompt(s)))
	if err != nil {
		return nil, err
	}
	for _, f := range findings {
		if f != nil {
			gologger.Info().Msgf("[LLM Behavior Analysis] %s", f.Name)
		}
	}
	return findings, nil
}

func behavioralAgent(ctx context.Context, s snapshot, f *llm.Factory) ([]*BehavioralRiskItem, error) {
	gologger.Info().Msgf("behavioral_analysis: start, skill_dir=%s", s.SkillDir)
	instruction, err := assets.Prompt("behavioral_analysis_system.md")
	if err != nil {
		return nil, err
	}
	submit, collector, err := llm.NewSubmitTool(llm.SubmitBehavTool,
		"Submit the behavioral analysis result with every risk you confirmed.")
	if err != nil {
		return nil, err
	}
	agent, _, err := f.NewAgent(ctx, "pro", llm.AgentOptions{
		Name:           "behavioral_analysis",
		Instruction:    instruction + fmt.Sprintf(llm.SubmitInstruction, llm.SubmitBehavTool),
		Tools:          []tool.BaseTool{submit},
		ReturnDirectly: []string{llm.SubmitBehavTool},
		// 保留 grep 与 read_file，禁掉写类与列目录类工具。
		HiddenTools: []string{"edit_file", "write_file", "ls", "glob"},
		BackendRoot: s.Workspace,
		Allow: []string{
			"/" + s.SkillName,
			// BehavioralPrompt 里给模型的是这个文档的完整路径，白名单与提示词
			// 必须一致，否则模型读到的是一次"路径不允许"。
			helperSkillDir,
		},
		SkillFS:  assets.SkillsFS(),
		MaxIters: maxModelCalls,
	})
	if err != nil {
		return nil, err
	}
	if _, runErr := RunAgent(ctx, agent, BehavioralPrompt(s)); runErr != nil && !isSoftStop(runErr) {
		return nil, runErr
	}
	findings, ok := ParseBehavioralFindings(collector.Raw())
	if !ok {
		// 行为分析没有可退回的原始命中，只能认下"这轮没有发现"；但必须说出来，
		// 否则"模型没按格式提交"会伪装成"这个 SKILL 没问题"。
		gologger.Warning().Msgf("behavioral_analysis: no structured result for %s; no behavioral findings", s.SkillName)
	}
	for _, f := range findings {
		if f != nil {
			gologger.Info().Msgf("[LLM Behavior Analysis] %s", f.Name)
		}
	}
	return findings, nil
}

// isSoftStop 把"模型调用耗尽"当成提前结束：eino 抛错会让整轮结果作废，但耗尽
// 之前已经提交的结果仍然有效，所以这里继续走解析而不是整体失败。
func isSoftStop(err error) bool {
	return errors.Is(err, adk.ErrExceedMaxIterations) || errors.Is(err, adk.ErrExceedMaxRetries)
}

func behavioralSchema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "risk_found": {"type": "boolean"},
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "category": {"type": "string", "enum": ["Remote Execution","Data Exfiltration","Persistence","Destructive","Obfuscation","Command Injection","Privilege Escalation","Sensitive File Access","Network Abuse","Prompt Injection"]},
          "severity": {"type": "string", "enum": ["low","medium","high","critical"]},
          "file_path": {"type": "string"},
          "line_number": {"type": "integer"},
          "name": {"type": "string"},
          "description": {"type": "string"}
        },
        "required": ["category","severity","file_path","name","description"]
      }
    }
  },
  "required": ["risk_found","findings"]
}`)
}
