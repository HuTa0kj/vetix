package plugin

import (
	"fmt"
	"regexp"
)

var reverseShellRe = regexp.MustCompile(`(?i)(/dev/(tcp|udp)/|\bnc\s+.*\s-e\s+|\bsocat\b.*\bexec:)`)

type ReverseShellPlugin struct{}

func (ReverseShellPlugin) Meta() Meta {
	return Meta{
		ID:          "reverse_shell",
		Name:        "Reverse Shell",
		Description: "Detects reverse-shell patterns such as /dev/tcp redirections, nc -e and socat exec:.",
	}
}

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
