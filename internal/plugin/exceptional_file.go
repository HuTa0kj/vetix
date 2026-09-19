package plugin

import "vetix/internal/pluginutils"

type ExceptionalFileCheckPlugin struct{}

func (ExceptionalFileCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "exceptional_file",
		Name:        "Non-Printable Characters",
		Description: "Flags text files containing a large share of non-printable or control characters.",
	}
}

func (ExceptionalFileCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	if !pluginutils.ExistNonText([]byte(content)) {
		return nil
	}
	return []Issue{{
		Name:          "Exceptional file",
		Severity:      SeverityMedium,
		Category:      CatObfuscation,
		Description:   "A large number of abnormal characters were found in a file that should have been readable.",
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}
