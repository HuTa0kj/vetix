package plugin

import "vetix/internal/pluginutils"

type RareFileCheckPlugin struct{}

func (RareFileCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "rare_file",
		Name:        "Rare File Extension",
		Description: "Flags files whose extension is outside the known text-file whitelist.",
	}
}

func (RareFileCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	if !pluginutils.IsRiskFile(filePath) {
		return nil
	}
	return []Issue{{
		Name:          "Rare file",
		Severity:      SeverityMedium,
		Category:      CatObfuscation,
		Description:   "The SKILL directory contains rare auxiliary files that may pose a security risk.",
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}
