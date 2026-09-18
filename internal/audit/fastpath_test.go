package audit

import (
	"testing"

	"github.com/cloudwego/eino/schema"
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

type errString string

func (e errString) Error() string { return string(e) }
