// Package jsonx 提供容错 JSON 解析兜底。
//
// 弱模型经工具调用返回结构化结果时经常给出畸形 JSON：包在 ``` 围栏里、带前后
// 说明文字、尾随逗号、未闭合的括号。这里逐级放宽，尽量把内容捞回来。
package jsonx

import (
	"encoding/json"
	"errors"
	"strings"
)

// Unmarshal 先严格解析，失败后走 Repair。
func Unmarshal(data []byte, v any) error {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return errors.New("empty JSON payload")
	}
	if err := json.Unmarshal([]byte(s), v); err == nil {
		return nil
	}
	repaired := Repair(s)
	if repaired == s {
		return json.Unmarshal([]byte(s), v)
	}
	if err := json.Unmarshal([]byte(repaired), v); err != nil {
		return err
	}
	return nil
}

// Repair 尽力把畸形 JSON 修成可解析文本。修不动就原样返回，交给调用方报错。
func Repair(s string) string {
	s = stripFences(s)
	s = extractOutermost(s)
	s = trimTrailingCommas(s)
	s = balanceBrackets(s)
	return s
}

func stripFences(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	t = strings.TrimPrefix(t, "```")
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		t = t[i+1:]
	}
	if i := strings.LastIndex(t, "```"); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// extractOutermost 截取从第一个 { 或 [ 到最后一个匹配的 } 或 ] 之间的内容，
// 丢弃模型附带的前后解释文字。
func extractOutermost(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return t
	}
	objStart := strings.IndexByte(t, '{')
	arrStart := strings.IndexByte(t, '[')
	var open, closeCh byte
	switch {
	case objStart < 0 && arrStart < 0:
		return t
	case arrStart < 0 || (objStart >= 0 && objStart < arrStart):
		open, closeCh = '{', '}'
	default:
		open, closeCh = '[', ']'
	}
	start := strings.IndexByte(t, open)
	end := strings.LastIndexByte(t, closeCh)
	if end < start {
		return t[start:]
	}
	return t[start : end+1]
}

// trimTrailingCommas 去掉对象/数组里最后一个元素后的逗号。
func trimTrailingCommas(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			sb.WriteByte(c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			sb.WriteByte(c)
		case ',':
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\n' || s[j] == '\r' || s[j] == '\t') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// balanceBrackets 补全未闭合的括号与字符串引号（截断的输出很常见）。
func balanceBrackets(s string) string {
	var stack []byte
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 && stack[len(stack)-1] == c {
				stack = stack[:len(stack)-1]
			}
		}
	}
	var sb strings.Builder
	sb.WriteString(s)
	if inStr {
		sb.WriteByte('"')
	}
	for i := len(stack) - 1; i >= 0; i-- {
		sb.WriteByte(stack[i])
	}
	return sb.String()
}
