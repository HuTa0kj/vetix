package plugin

import (
	"fmt"
	"regexp"
)

// 每条规则锚定一种"下载即执行"的签名：curl/wget 经管道送入 shell 或脚本解释器、
// PowerShell 下载管道进 Invoke-Expression、进程替换/eval/source 直接消费下载流。
// 执行的是下载时刻的任意代码，审查时看到的安装命令和实际运行的内容可以完全不同。
var remoteScriptRules = []struct {
	label   string
	pattern *regexp.Regexp
}{
	{"download piped into a shell interpreter", regexp.MustCompile(`(?i)\b(?:curl|wget)\b[^|\n]*\|\s*(?:sudo\s+)?(?:ba|z|da|fi)?sh\b`)},
	{"download piped into a scripting language", regexp.MustCompile(`(?i)\b(?:curl|wget)\b[^|\n]*\|\s*(?:sudo\s+)?(?:python[0-9.]*|perl|ruby|node|deno|bun)\b`)},
	{"PowerShell download piped into Invoke-Expression", regexp.MustCompile(`(?i)\b(?:iwr|irm|invoke-webrequest|invoke-restmethod)\b[^|\n]*\|\s*(?:iex|invoke-expression)\b`)},
	{"download consumed via process substitution", regexp.MustCompile(`(?i)\b(?:ba|z|da|fi)?sh\s+<\s*\(\s*(?:curl|wget)\b`)},
	{"shell -c wrapping an inline download", regexp.MustCompile(`(?i)\b(?:ba|z|da|fi)?sh\s+-c\s*['"][^'"\n]*\b(?:curl|wget)\b`)},
	{"eval of a download", regexp.MustCompile(`(?i)\beval\s*['"\s]*\$?\(\s*(?:curl|wget)\b`)},
	{"source of a download", regexp.MustCompile(`(?i)\b(?:source|\.)\s*<\s*\(\s*(?:curl|wget)\b`)},
}

type RemoteScriptExecCheckPlugin struct{}

func (RemoteScriptExecCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "remote_script_exec",
		Name:        "Remote Script Execution",
		Description: "Detects download-and-execute signatures — curl or wget piped into bash or sh, PowerShell downloads piped into Invoke-Expression, process substitutions and eval or source of downloaded streams.",
	}
}

func (RemoteScriptExecCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	// 同一条规则在同一行去重：同一行反复出现同类签名只报一条。
	seen := map[string]bool{}
	for i, rule := range remoteScriptRules {
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
				Name:     "Remote script execution",
				Severity: SeverityCritical,
				// 文档里引用合法工具的官方安装命令也会命中，上下文交给复核判断。
				Category:      CatRemoteExecution,
				Description:   fmt.Sprintf("Detected %s (%s).", rule.label, text),
				FilePath:      rel,
				Line:          line,
				AuditRequired: true,
			})
		}
	}
	return issues
}
