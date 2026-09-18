// Package pytext 复刻 Python 文本格式化，只用于构造喂给模型的提示词。
//
// 这些格式看起来很脏（单引号、多一层根、枚举的 <Severity.CRITICAL: 'critical'>
// 写法），但既有提示词就是在这套文本上调优的，改动会让模型行为无迹可循。
package pytext

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Quote 复刻 Python 的 repr() 对字符串的处理。
func Quote(s string) string {
	var sb strings.Builder
	sb.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'':
			sb.WriteString(`\'`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('\'')
	return sb.String()
}

// StrDict 渲染单层字典，键按 Python 的插入顺序不可复现，这里固定按字典序，
// 保证同一输入的提示词文本稳定。
func StrDict(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(Quote(k))
		sb.WriteString(": ")
		sb.WriteString(Value(m[k]))
	}
	sb.WriteByte('}')
	return sb.String()
}

func Value(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case string:
		return Quote(t)
	case bool:
		if t {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case map[string]any:
		return StrDict(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			parts = append(parts, Value(e))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// EnumValue 复刻 Python Enum 的 repr，例如 <Severity.CRITICAL: 'critical'>。
func EnumValue(enumName, member, value string) string {
	return fmt.Sprintf("<%s.%s: %s>", enumName, member, Quote(value))
}

func List(items []string) string {
	return "[" + strings.Join(items, ", ") + "]"
}
