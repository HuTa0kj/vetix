package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/projectdiscovery/gologger"

	"vetix/internal/config"
	"vetix/internal/jsonx"
	"vetix/internal/llm"
)

// 单文件快速路径拿结构化输出的候选策略。
//
// 网关对结构化输出的支持差异很大，尤其是"思考模式 + 强制工具调用"这个组合：
// 实测有网关直接拒绝，返回 "Thinking mode does not support this tool_choice"。
// 所以这里不假定某一种方案可用，而是按优先级依次尝试，被拒就降级到下一档。
const (
	strategyForcedTool  = "forced tool call"
	strategyJSONSchema  = "response_format json_schema"
	strategyJSONObject  = "response_format json_object"
	jsonOnlyInstruction = "\n\nRespond with a single valid JSON object and nothing else. Do not wrap it in prose or code fences."
)

type structuredAttempt struct {
	name string
	run  func(context.Context) (*schema.Message, error)
	// parse 返回 ok=false 表示这一档没拿到结构化结果，应当继续降级。
	parse func(*schema.Message) ([]*BehavioralRiskItem, bool)
}

// fastPathAttempts 按优先级给出候选策略。
//
// 顺序取决于思考模式：开启思考时强制工具调用在部分网关上被拒，因此把它放到最后；
// 关闭思考时它是最可靠的（schema 由工具定义强约束），放第一位。配置里显式写了
// response_format: json_schema 的，尊重用户选择优先尝试 schema 约束。
func fastPathAttempts(cm *openai.ChatModel, m *config.Model, system, user string) []structuredAttempt {
	msgs := []*schema.Message{schema.SystemMessage(system), schema.UserMessage(user)}
	jsonMsgs := []*schema.Message{schema.SystemMessage(system), schema.UserMessage(user + jsonOnlyInstruction)}

	forcedTool := structuredAttempt{
		name: strategyForcedTool,
		run: func(ctx context.Context) (*schema.Message, error) {
			info := llm.BehavioralToolInfo()
			if info == nil {
				return nil, errors.New("behavioral submit tool schema is unavailable")
			}
			tcm, err := cm.WithTools([]*schema.ToolInfo{info})
			if err != nil {
				return nil, err
			}
			return tcm.Generate(ctx, msgs, model.WithToolChoice(schema.ToolChoiceForced, llm.SubmitBehavTool))
		},
		parse: func(msg *schema.Message) ([]*BehavioralRiskItem, bool) {
			if msg == nil {
				return nil, false
			}
			for _, tc := range msg.ToolCalls {
				if tc.Function.Name != llm.SubmitBehavTool {
					continue
				}
				// 参数解析失败也算这一档没拿到结构化结果，不能当成"模型确认零风险"。
				return ParseBehavioralFindings(tc.Function.Arguments)
			}
			return nil, false
		},
	}

	jsonSchema := structuredAttempt{
		name: strategyJSONSchema,
		run: func(ctx context.Context) (*schema.Message, error) {
			return cm.Generate(ctx, jsonMsgs, llm.WithResponseFormat(llm.SubmitBehavTool, behavioralSchema()))
		},
		parse: parseBehavioralContentStrict,
	}

	jsonObject := structuredAttempt{
		name: strategyJSONObject,
		run: func(ctx context.Context) (*schema.Message, error) {
			return cm.Generate(ctx, jsonMsgs, llm.WithJSONObjectFormat())
		},
		parse: parseBehavioralContentStrict,
	}

	thinking := m.Thinking != nil && *m.Thinking
	if thinking || m.ResponseFormat == "json_schema" {
		return []structuredAttempt{jsonSchema, jsonObject, forcedTool}
	}
	return []structuredAttempt{forcedTool, jsonSchema, jsonObject}
}

// parseBehavioralContentStrict 只有在响应里真的出现了 findings 字段时才算成功。
// 否则无法区分"模型正常返回了空结果"和"模型回了段散文"，会误判成解析成功。
func parseBehavioralContentStrict(msg *schema.Message) ([]*BehavioralRiskItem, bool) {
	if msg == nil || strings.TrimSpace(msg.Content) == "" {
		return nil, false
	}
	// 先取字段是否存在，再解字段本身：probe["findings"] 是数组，必须解进 slice，
	// 解进带 Findings 字段的结构体会直接失败。
	var probe map[string]json.RawMessage
	if err := jsonx.Unmarshal([]byte(msg.Content), &probe); err != nil {
		return nil, false
	}
	raw, ok := probe["findings"]
	if !ok {
		return nil, false
	}
	var findings []*BehavioralRiskItem
	if err := json.Unmarshal(raw, &findings); err != nil {
		return nil, false
	}
	return findings, true
}

// shouldFallBack 判断这个错误是否值得换一档策略重试。
//
// 判据取"除鉴权/网络外都重试"而不是精确匹配状态码：各网关拒绝请求形状时的措辞
// 五花八门（"Thinking mode does not support this tool_choice"、schema 不合法、
// 参数不支持……），逐个枚举必然会漏。代价是真正的配置错误会多打两次请求，但最终
// 仍会把原错误上抛，用户看到的还是根因。鉴权失败重试没有意义，直接返回。
func shouldFallBack(err error) bool {
	s := strings.ToLower(err.Error())
	for _, fatal := range []string{"status code: 401", "status code: 403", "invalid token", "unauthorized", "no such host", "connection refused"} {
		if strings.Contains(s, fatal) {
			return false
		}
	}
	return true
}

// runAttempts 依次尝试各档策略。任何一档产出结构化结果就返回；被网关以请求形状
// 拒绝时降级到下一档并记录原因，最后一档的请求级失败才上抛。
//
// 注意"请求成功但没拿到结构化输出"（模型回了散文）不算失败：它可能是后续档位能救
// 回来的情况，也可能是最后一档的结局。后者按"没有行为发现"收尾，而不是把前面档位
// 那个已被降级掉的错误翻出来上抛——那个错误已经不代表最终状态了，上抛它会让一次本
// 可以出报告的扫描整体失败。
func runAttempts(ctx context.Context, attempts []structuredAttempt) ([]*BehavioralRiskItem, error) {
	for i, at := range attempts {
		msg, err := at.run(ctx)
		if err != nil {
			if i == len(attempts)-1 || !shouldFallBack(err) {
				return nil, err
			}
			gologger.Warning().Msgf("single_file_analysis: %s failed (%v); falling back to %s",
				at.name, err, attempts[i+1].name)
			continue
		}
		findings, ok := at.parse(msg)
		if ok {
			return findings, nil
		}
		if i < len(attempts)-1 {
			gologger.Warning().Msgf("single_file_analysis: %s produced no structured output; falling back to %s",
				at.name, attempts[i+1].name)
		}
	}
	gologger.Warning().Msg("single_file_analysis: every structured output strategy failed; no behavioral findings")
	return nil, nil
}
