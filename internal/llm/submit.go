package llm

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// 提交工具的名字直接取 Python 侧结构化输出的类名。这个约定来自 LangChain：
// with_structured_output 的实现就是造一个以 schema 类名命名的伪工具，json_schema
// 模式下该名字也会作为 schema 名下发给模型。
const (
	SubmitVerifyTool = "PluginsVerificationResult"
	SubmitBehavTool  = "BehavioralAnalysisResult"
)

// SubmitInstruction 是相对 Python 版必须补的一段话。
//
// Python 用 LangChain 的 ToolStrategy，由框架负责让模型知道要调用提交工具；eino
// 没有等价机制，工具只是出现在工具列表里。在不动既有提示词文件的前提下，只能靠
// Instruction 补说明，否则模型探索完会直接输出自由文本而不提交。
const SubmitInstruction = "\n\nWhen you have finished the analysis, call the `%s` tool exactly once with all confirmed findings. Do not output the result as plain text."

// Collector 保存提交工具收到的原始参数 JSON。
//
// 不从事件流里取参数：return-direct 的终态事件携带的是工具的字符串返回值而不是
// 调用参数，参数在更早的 assistant 消息上。让工具自己回写参数最稳。
type Collector struct {
	mu  sync.Mutex
	raw string
}

func (c *Collector) Raw() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.raw
}

func (c *Collector) set(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.raw = s
}

type verifySubmitArgs struct {
	Findings []RiskFinding `json:"findings"`
}

type behavioralSubmitArgs struct {
	RiskFound bool                  `json:"risk_found"`
	Findings  []*BehavioralRiskItem `json:"findings"`
}

// NewSubmitTool 生成提交工具：输入结构体本身就是结构化输出的 schema。
func NewSubmitTool(name, desc string) (tool.BaseTool, *Collector, error) {
	c := &Collector{}
	var t tool.InvokableTool
	var err error
	switch name {
	case SubmitVerifyTool:
		t, err = utils.InferTool(name, desc, func(ctx context.Context, in verifySubmitArgs) (string, error) {
			return "ok", nil
		})
	case SubmitBehavTool:
		t, err = utils.InferTool(name, desc, func(ctx context.Context, in behavioralSubmitArgs) (string, error) {
			return "ok", nil
		})
	default:
		return nil, nil, &unknownToolError{name: name}
	}
	if err != nil {
		return nil, nil, err
	}
	return &collectingTool{InvokableTool: t, c: c}, c, nil
}

type unknownToolError struct{ name string }

func (e *unknownToolError) Error() string { return "unknown submit tool: " + e.name }

// collectingTool 先留存原始参数串，再交给真正的工具实现。
type collectingTool struct {
	tool.InvokableTool
	c *Collector
}

func (t *collectingTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	t.c.set(argumentsInJSON)
	return t.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
}

// BehavioralToolInfo 暴露行为分析提交工具的 schema，供单文件快速路径在无 deep
// agent 的情况下直接以工具调用形式取结构化输出。
func BehavioralToolInfo() *schema.ToolInfo {
	t, err := utils.InferTool(SubmitBehavTool, "Submit the behavioral analysis result.", func(ctx context.Context, in behavioralSubmitArgs) (string, error) {
		return "ok", nil
	})
	if err != nil {
		return nil
	}
	info, err := t.Info(context.Background())
	if err != nil {
		return nil
	}
	return info
}

// WithResponseFormat 按调用注入 response_format。eino 只在构造期支持
// ResponseFormat，所以走请求体改写：这个 hook 在请求序列化之后执行。
func WithResponseFormat(schemaName string, raw json.RawMessage) model.Option {
	var parsed any
	_ = json.Unmarshal(raw, &parsed)
	return withResponseFormat(map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   schemaName,
			"schema": parsed,
		},
	})
}

// WithJSONObjectFormat 只要求网关返回合法 JSON，不约束 schema。比 json_schema 弱，
// 但兼容性最好，作为 schema 被网关拒绝时的下一档。
func WithJSONObjectFormat() model.Option {
	return withResponseFormat(map[string]any{"type": "json_object"})
}

func withResponseFormat(rf map[string]any) model.Option {
	return openai.WithRequestPayloadModifier(func(ctx context.Context, _ []*schema.Message, body []byte) ([]byte, error) {
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, err
		}
		m["response_format"] = rf
		return json.Marshal(m)
	})
}
