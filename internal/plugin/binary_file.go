package plugin

import "vetix/internal/pluginutils"

type BinaryFileCheckPlugin struct{}

func (BinaryFileCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "binary_file",
		Name:        "Binary File",
		Description: "Flags binary files inside the SKILL directory, where payloads are invisible to text-based analysis.",
	}
}

func (BinaryFileCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	if !pluginutils.IsBinaryFile(filePath) {
		return nil
	}
	return []Issue{{
		Name:          "Binary file",
		Severity:      SeverityHigh,
		Category:      CatObfuscation,
		Description:   "Suspicious binary files were found in the SKILL directory.",
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}
