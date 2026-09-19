package plugin

import "strings"

type ConsecutiveNewlinesCheckPlugin struct{}

func (ConsecutiveNewlinesCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "consecutive_newlines",
		Name:        "Excessive Consecutive Newlines",
		Description: "Detects runs of 30+ consecutive newlines used to push commands out of the visible terminal viewport.",
	}
}

func (ConsecutiveNewlinesCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	if !strings.Contains(content, strings.Repeat("\n", 30)) {
		return nil
	}
	return []Issue{{
		Name:     "Large number of consecutive line breaks",
		Severity: SeverityHigh,
		Category: CatObfuscation,
		Description: "The file contains a large number of consecutive newline characters, " +
			"which may indicate the presence of malicious commands behind the newlines.",
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}
