package jsonx

import "testing"

// 弱模型给出的结构化输出经常是畸形 JSON，这一层是 Python 版 json_repair 的替代。
func TestUnmarshalRepairs(t *testing.T) {
	type payload struct {
		Findings []struct {
			Name     string `json:"name"`
			Severity string `json:"severity"`
		} `json:"findings"`
	}

	cases := map[string]string{
		"fenced":         "```json\n{\"findings\":[{\"name\":\"n\",\"severity\":\"high\"}]}\n```",
		"prose around":   "Here is the result:\n{\"findings\":[{\"name\":\"n\",\"severity\":\"high\"}]}\nHope this helps.",
		"trailing comma": `{"findings":[{"name":"n","severity":"high"},]}`,
		"truncated":      `{"findings":[{"name":"n","severity":"high"}`,
		"already valid":  `{"findings":[{"name":"n","severity":"high"}]}`,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			var out payload
			if err := Unmarshal([]byte(in), &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(out.Findings) != 1 || out.Findings[0].Severity != "high" {
				t.Fatalf("unexpected result: %+v", out)
			}
		})
	}

	// 数组是否也被提取出来。修复不了时必须报错，不能静默返回空结构。
	var arr []string
	if err := Unmarshal([]byte("blah [\"a\",\"b\"] trailing"), &arr); err != nil || len(arr) != 2 {
		t.Fatalf("array extraction: %v %v", arr, err)
	}
	var bad map[string]any
	if err := Unmarshal([]byte("not json at all"), &bad); err == nil {
		t.Fatal("garbage input must fail rather than silently succeed")
	}
	if err := Unmarshal(nil, &bad); err == nil {
		t.Fatal("empty input must fail")
	}
}
