package plugin

import (
	"fmt"
	"strings"

	"vetix/internal/pluginutils"
)

type LargeFileCheckPlugin struct{}

const largeFileMaxBytes = 2 * 1024 * 1024

func (LargeFileCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	// content 是原始字节的原样字符串转换，len 与磁盘上的文件大小一致。以读入的
	// 内容为准而不是 stat，插件只对被扫描到的内容负责。
	size := len(content)
	if size <= largeFileMaxBytes {
		return nil
	}
	return []Issue{{
		Name:     "Large SKILL file found",
		Severity: SeverityMedium,
		Category: CatObfuscation,
		Description: fmt.Sprintf("SKILL file size %s exceeds limit of %s.",
			pluginutils.HumanBytes(size), pluginutils.HumanBytes(largeFileMaxBytes)),
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: false,
	}}
}

type LongFileCheckPlugin struct{}

const longFileMaxLines = 3000

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

type BinaryFileCheckPlugin struct{}

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

type ConsecutiveNewlinesCheckPlugin struct{}

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

type ExceptionalFileCheckPlugin struct{}

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

type RareFileCheckPlugin struct{}

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
