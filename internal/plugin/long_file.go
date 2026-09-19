package plugin

import (
	"fmt"

	"vetix/internal/pluginutils"
)

const longFileMaxLines = 3000

type LongFileCheckPlugin struct{}

func (LongFileCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "long_file",
		Name:        "Extremely Long File",
		Description: "Flags single files over 3000 lines, enough to hide content beyond what a reviewer will actually read.",
	}
}

func (LongFileCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	lines := len(pluginutils.SplitLines(content))
	if lines <= longFileMaxLines {
		return nil
	}
	return []Issue{{
		Name:     "Extremely long file",
		Severity: SeverityMedium,
		Category: CatObfuscation,
		Description: fmt.Sprintf(
			"An excessively long file, totaling %d lines, was found in the SKILL directory.", lines),
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}
