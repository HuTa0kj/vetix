package audit

import (
	"context"
	"encoding/json"

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

// ParseVerifyFindings 解析复核结果。第二个返回值区分"解出来了"与"没解出来"：前者
// 即使 findings 为空也是有效结论（模型判定全部是误报），后者必须由调用方兜底——把
// 两者混为一谈会让需要复核的命中静默消失。
//
// findings 字段必须真实存在才承认解析成功：json-repair 会把垃圾参数洗成 `{"key":""}`
// 这类凭空造出来的合法 JSON，直接解进结构体会得到零值，冒充"模型确认零风险"。
func ParseVerifyFindings(raw string) ([]RiskFinding, bool) {
	if raw == "" {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if err := jsonx.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, false
	}
	if _, ok := probe["findings"]; !ok {
		return nil, false
	}
	var out struct {
		Findings []RiskFinding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return out.Findings, true
}

// ParseBehavioralFindings 同理，解析失败返回 ok=false。findings 字段存在性探测的
// 理由见 ParseVerifyFindings。
func ParseBehavioralFindings(raw string) ([]*BehavioralRiskItem, bool) {
	if raw == "" {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if err := jsonx.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, false
	}
	if _, ok := probe["findings"]; !ok {
		return nil, false
	}
	var out struct {
		RiskFound bool                  `json:"risk_found"`
		Findings  []*BehavioralRiskItem `json:"findings"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return out.Findings, true
}
