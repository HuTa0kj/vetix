package plugin

import (
	"fmt"

	"vetix/internal/pluginutils"
)

const largeFileMaxBytes = 2 * 1024 * 1024

type LargeFileCheckPlugin struct{}

func (LargeFileCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "large_file",
		Name:        "Oversized File",
		Description: "Flags single files over 2 MB, a size at which obfuscated content is easy to bury.",
	}
}

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
