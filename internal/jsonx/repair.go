// Package jsonx 提供容错 JSON 解析兜底。
//
// 弱模型经工具调用返回结构化结果时经常给出畸形 JSON：包在 ``` 围栏里、带前后
// 说明文字、尾随逗号、未闭合的括号。修复交给 json-repair，修不动就报错。
package jsonx

import (
	"encoding/json"
	"errors"
	"strings"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
)

// Unmarshal 先严格解析，失败后走修复再解析。
func Unmarshal(data []byte, v any) error {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return errors.New("empty JSON payload")
	}
	if err := json.Unmarshal([]byte(s), v); err == nil {
		return nil
	}
	repaired, err := jsonrepair.RepairJSON(s)
	if err != nil {
		return json.Unmarshal([]byte(s), v)
	}
	return json.Unmarshal([]byte(repaired), v)
}
