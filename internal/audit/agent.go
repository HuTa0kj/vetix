package audit

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"vetix/internal/jsonx"
)

// RunAgent 跑一轮 agent，返回最后一个 assistant 消息。
func RunAgent(ctx context.Context, agent adk.ResumableAgent, prompt string) (*schema.Message, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	it := runner.Run(ctx, []*schema.Message{schema.UserMessage(prompt)})

	var last *schema.Message
	for {
		ev, ok := it.Next()
		if !ok {
			break
		}
		if ev.Err != nil {
			return last, ev.Err
		}
		msg, _, err := adk.GetMessage(ev)
		if err != nil || msg == nil {
			continue
		}
		if msg.Role == schema.Assistant {
			last = msg
		}
	}
	return last, nil
}

// ParseVerifyFindings 解析复核结果。解析不出来时返回 nil：Python 版在结构化输出
// 拿不到时会退回"只保留无需复核的命中"，这里保持同样的降级而不是报错。
func ParseVerifyFindings(raw string) []RiskFinding {
	if raw == "" {
		return nil
	}
	var out struct {
		Findings []RiskFinding `json:"findings"`
	}
	if err := jsonx.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out.Findings
}

// ParseBehavioralFindings 同理，解析失败返回 nil。
func ParseBehavioralFindings(raw string) []*BehavioralRiskItem {
	if raw == "" {
		return nil
	}
	var out struct {
		RiskFound bool                  `json:"risk_found"`
		Findings  []*BehavioralRiskItem `json:"findings"`
	}
	if err := jsonx.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out.Findings
}
