package plugin

import (
	"fmt"
	"regexp"
)

// 每条规则锚定一种用反射拆散危险函数名的写法：getattr 以字符串字面量引用
// exec / eval / system 等 sink、对危险模块做动态 getattr、importlib.import_module
// 与 __import__ 接变量参数、globals()/locals() 取 __builtins__。sink 名对大小写
// 敏感（Python 语义如此），故意不加 (?i)。getattr 的字符串实参若是无害属性
// （如 os.environ）、import_module 的实参若是字面量，都不算命中。
var reflectiveRules = []struct {
	label   string
	pattern *regexp.Regexp
}{
	// getattr 引用危险 sink 名
	{"getattr with dangerous sink", regexp.MustCompile(`\bgetattr\s*\(\s*[^,()\n]{1,40},\s*['"](system|popen|exec|execfile|eval|compile|__import__|spawn[lv][pe]?|fork)['"]\s*[,)]`)},
	// 对危险模块做动态 getattr（第二参数不是字符串字面量）
	{"dynamic getattr on dangerous module", regexp.MustCompile(`\bgetattr\s*\(\s*(?:os|sys|builtins|importlib|subprocess|shutil|ctypes)\s*,\s*[^'"\s,)\d]`)},
	// 动态 import_module（参数不是字符串字面量）
	{"dynamic import_module", regexp.MustCompile(`\bimport_module\s*\(\s*[^'"\s,)]`)},
	// 动态 __import__（参数不是字符串字面量）
	{"dynamic __import__", regexp.MustCompile(`\b__import__\s*\(\s*[^'"\s,)]`)},
	// 经 globals()/locals() 拿 __builtins__ / __import__
	{"builtins via globals or locals", regexp.MustCompile(`\b(?:globals|locals)\s*\(\s*\)\s*\[\s*['"]__(?:builtins|import)__['"]`)},
}

type ReflectiveCallCheckPlugin struct{}

func (ReflectiveCallCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "reflective_call",
		Name:        "Reflective Calls",
		Description: "Detects reflective access used to split dangerous function names apart — getattr(os, 'system'), dynamic __import__ or import_module, builtins reached through globals or locals.",
	}
}

func (ReflectiveCallCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	// 同一条规则在同一行去重：同一行反复出现同类反射写法只报一条。
	seen := map[string]bool{}
	for i, rule := range reflectiveRules {
		for _, m := range rule.pattern.FindAllStringIndex(content, -1) {
			line := lineOf(content, m[0])
			key := fmt.Sprintf("%d:%d", i, line)
			if seen[key] {
				continue
			}
			seen[key] = true
			text := content[m[0]:m[1]]
			// 按字符截断，避免把多字节 UTF-8 字符切成半个码点。
			if runes := []rune(text); len(runes) > 80 {
				text = string(runes[:80]) + "..."
			}
			issues = append(issues, Issue{
				Name:          "Reflective call",
				Severity:      SeverityHigh,
				Category:      CatObfuscation,
				Description:   fmt.Sprintf("References %s (%s).", rule.label, text),
				FilePath:      rel,
				Line:          line,
				AuditRequired: true,
			})
		}
	}
	return issues
}
