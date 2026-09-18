package audit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"vetix/internal/config"
	"vetix/internal/llm"
	"vetix/internal/plugin"
)

// 单文件快速路径解析结构化输出。曾经把 findings 数组解进带 Findings 字段的结构体，
// 导致合法 JSON 也被判为失败并一路降级，最后整轮报错。
func TestParseBehavioralContentStrict(t *testing.T) {
	valid := `{"risk_found":true,"findings":[{"category":"Obfuscation","severity":"medium",` +
		`"file_path":"SKILL.md","line_number":5,"name":"n","description":"d"}]}`

	findings, ok := parseBehavioralContentStrict(schema.AssistantMessage(valid, nil))
	if !ok || len(findings) != 1 {
		t.Fatalf("valid payload must parse: ok=%v findings=%d", ok, len(findings))
	}
	if findings[0].LineNumber != 5 || findings[0].Category != "Obfuscation" {
		t.Fatalf("fields not decoded: %+v", findings[0])
	}

	// 空结果也算解析成功——"模型确认没有风险"和"模型没按格式回答"必须区分开。
	if findings, ok := parseBehavioralContentStrict(schema.AssistantMessage(`{"risk_found":false,"findings":[]}`, nil)); !ok || len(findings) != 0 {
		t.Errorf("empty findings must still be a success: ok=%v", ok)
	}

	for name, content := range map[string]string{
		"prose":        "I found no issues.",
		"missing key":  `{"risk_found":false}`,
		"empty":        "",
		"fenced prose": "```json\nnot really json\n```",
	} {
		if _, ok := parseBehavioralContentStrict(schema.AssistantMessage(content, nil)); ok {
			t.Errorf("%s must not parse as structured output", name)
		}
	}
	if _, ok := parseBehavioralContentStrict(nil); ok {
		t.Error("nil message must not parse")
	}
}

func TestShouldFallBack(t *testing.T) {
	// 请求形状被拒值得换一档；鉴权/网络问题重试没有意义。
	mustFallBack := []string{
		"error, status code: 400, status: 400 Bad Request, message: Thinking mode does not support this tool_choice",
		"error, status code: 422, status: Unprocessable Entity, message: bad schema",
	}
	for _, s := range mustFallBack {
		if !shouldFallBack(errString(s)) {
			t.Errorf("should fall back: %s", s)
		}
	}
	mustNot := []string{
		"error, status code: 401, status: 401 Unauthorized, message: Invalid token",
		"dial tcp: lookup router.example: no such host",
		"dial tcp 127.0.0.1:1: connect: connection refused",
	}
	for _, s := range mustNot {
		if shouldFallBack(errString(s)) {
			t.Errorf("must not fall back: %s", s)
		}
	}
}

// 前面档位被网关拒过、后面档位请求成功但没给出结构化输出时，不能把那个已经被降级
// 掉的错误当成最终结果上抛——那会让一次本可以出报告的扫描整体失败，而错误信息还指
// 向一个早就放弃了的档位。
func TestRunAttemptsDoesNotResurrectStaleError(t *testing.T) {
	prose := func(context.Context) (*schema.Message, error) {
		return schema.AssistantMessage("I found no issues.", nil), nil
	}
	attempts := []structuredAttempt{
		{name: "rejected", run: func(context.Context) (*schema.Message, error) {
			return nil, errString("error, status code: 400, message: Thinking mode does not support this tool_choice")
		}, parse: func(*schema.Message) ([]*BehavioralRiskItem, bool) { return nil, false }},
		{name: "prose-1", run: prose, parse: parseBehavioralContentStrict},
		{name: "prose-2", run: prose, parse: parseBehavioralContentStrict},
	}

	findings, err := runAttempts(context.Background(), attempts)
	if err != nil {
		t.Fatalf("stale error from an already-downgraded attempt must not surface: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}

// 最后一档的请求级失败必须上抛：那是真实故障，不能降级成"没有发现"。
func TestRunAttemptsPropagatesLastRequestFailure(t *testing.T) {
	last := errors.New("connection reset by peer")
	attempts := []structuredAttempt{
		{name: "a0", run: func(context.Context) (*schema.Message, error) { return nil, last }, parse: parseBehavioralContentStrict},
	}
	if _, err := runAttempts(context.Background(), attempts); !errors.Is(err, last) {
		t.Fatalf("last attempt's failure must surface, got %v", err)
	}
}

// 鉴权类错误不降级，立刻上抛原因为用户可自行修正。
func TestRunAttemptsStopsOnFatalError(t *testing.T) {
	fatal := errString("error, status code: 401, message: Invalid token")
	ran := false
	attempts := []structuredAttempt{
		{name: "a0", run: func(context.Context) (*schema.Message, error) { return nil, fatal }, parse: parseBehavioralContentStrict},
		{name: "a1", run: func(context.Context) (*schema.Message, error) {
			ran = true
			return schema.AssistantMessage(`{"risk_found":false,"findings":[]}`, nil), nil
		}, parse: parseBehavioralContentStrict},
	}
	if _, err := runAttempts(context.Background(), attempts); err == nil {
		t.Fatal("auth failure must surface")
	}
	if ran {
		t.Error("auth failure must not fall back to the next strategy")
	}
}

// 强制工具调用档拿到 tool call 但参数是畸形 JSON 时，不能算作"模型确认零风险"。
func TestForcedToolParseRejectsMalformedArguments(t *testing.T) {
	attempts := fastPathAttempts(nil, &config.Model{}, "sys", "user")
	var forced structuredAttempt
	found := false
	for _, a := range attempts {
		if a.name == strategyForcedTool {
			forced, found = a, true
		}
	}
	if !found {
		t.Fatal("forced tool attempt missing")
	}
	msg := schema.AssistantMessage("", []schema.ToolCall{{
		Function: schema.FunctionCall{Name: llm.SubmitBehavTool, Arguments: "{not json"},
	}})
	if _, ok := forced.parse(msg); ok {
		t.Error("malformed tool arguments must not count as a parsed result")
	}
	if _, ok := forced.parse(nil); ok {
		t.Error("nil message must not count as a parsed result")
	}
	good := schema.AssistantMessage("", []schema.ToolCall{{
		Function: schema.FunctionCall{Name: llm.SubmitBehavTool, Arguments: `{"risk_found":false,"findings":[]}`},
	}})
	if got, ok := forced.parse(good); !ok || len(got) != 0 {
		t.Errorf("valid empty result must parse: ok=%v findings=%d", ok, len(got))
	}
}

// Parse* 的第二个返回值必须能区分"解出来了且为空"与"没解出来"，否则调用方无法在
// 解析失败时兜底。
func TestFindingsParsersReportSuccess(t *testing.T) {
	if _, ok := ParseVerifyFindings(`{"findings":[]}`); !ok {
		t.Error("empty findings is a valid result")
	}
	if _, ok := ParseVerifyFindings("totally not json"); ok {
		t.Error("garbage must not be reported as a valid result")
	}
	if _, ok := ParseVerifyFindings(""); ok {
		t.Error("empty payload must not be reported as a valid result")
	}
	if got, ok := ParseBehavioralFindings(`{"risk_found":true,"findings":[{"name":"n"}]}`); !ok || len(got) != 1 {
		t.Errorf("valid payload: ok=%v findings=%d", ok, len(got))
	}
	if _, ok := ParseBehavioralFindings("prose"); ok {
		t.Error("prose must not be reported as a valid result")
	}
}

// 复核拿不到结构化结果时要退回原始命中并标注未复核，而不是让这些命中消失。
func TestUnverifiedHitsAreKeptAndLabelled(t *testing.T) {
	got := unverifiedHits([]plugin.Issue{{
		Name: "Reverse shell", Description: "nc -e", Severity: plugin.SeverityCritical,
		Category: plugin.CatRemoteExecution, FilePath: "SKILL.md", Line: 7, AuditRequired: true,
	}})
	if len(got) != 1 {
		t.Fatalf("hits must be kept, got %d", len(got))
	}
	if got[0].Severity != "critical" || got[0].Line != 7 || got[0].FilePath != "SKILL.md" {
		t.Errorf("fields must survive: %+v", got[0])
	}
	if !strings.Contains(got[0].Description, "unverified") {
		t.Errorf("a degraded finding must say so: %q", got[0].Description)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
