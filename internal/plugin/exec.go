package plugin

import (
	"fmt"
	"regexp"
)

var (
	base64ExecRe   = regexp.MustCompile(`(?i)base64\s+(?:-d|--decode)[^|]*?\|\s*(ba)?sh`)
	reverseShellRe = regexp.MustCompile(`(?i)(/dev/(tcp|udp)/|\bnc\s+.*\s-e\s+|\bsocat\b.*\bexec:)`)
)

type Base64ExecPlugin struct{}

func (Base64ExecPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	for _, m := range base64ExecRe.FindAllStringIndex(content, -1) {
		issues = append(issues, Issue{
			Name:          "Base64 command piped to shell",
			Severity:      SeverityCritical,
			Category:      CatRemoteExecution,
			Description:   fmt.Sprintf("Base64-decoded command is piped into a shell (%s).", content[m[0]:m[1]]),
			FilePath:      rel,
			Line:          lineOf(content, m[0]),
			AuditRequired: true,
		})
	}
	return issues
}

type ReverseShellPlugin struct{}

func (ReverseShellPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	for _, m := range reverseShellRe.FindAllStringIndex(content, -1) {
		issues = append(issues, Issue{
			Name:          "Reverse shell",
			Severity:      SeverityCritical,
			Category:      CatRemoteExecution,
			Description:   fmt.Sprintf("Reverse shell pattern detected (%s).", content[m[0]:m[1]]),
			FilePath:      rel,
			Line:          lineOf(content, m[0]),
			AuditRequired: true,
		})
	}
	return issues
}
