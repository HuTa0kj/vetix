package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// RiskCategory 是报告里 category 字段的取值域，10 类。取值即报告中的字面文本，
// 改动会让历史报告无法对照。
const (
	CatRemoteExecution     = "Remote Execution"
	CatDataExfiltration    = "Data Exfiltration"
	CatPersistence         = "Persistence"
	CatDestructive         = "Destructive"
	CatObfuscation         = "Obfuscation"
	CatCommandInjection    = "Command Injection"
	CatPrivilegeEscalation = "Privilege Escalation"
	CatSensitiveFileAccess = "Sensitive File Access"
	CatNetworkAbuse        = "Network Abuse"
	CatPromptInjection     = "Prompt Injection"
)

type Issue struct {
	Name        string
	Description string
	Severity    Severity
	FilePath    string
	Category    string
	Line        int
	// AuditRequired 为 true 的命中要送 LLM 复核；false 的直接进入最终报告。
	AuditRequired bool
}

type Plugin interface {
	// Scan 返回 content 中的命中项；插件自行决定适用于哪些文件。
	Scan(skillDir, filePath, content string) []Issue
}

// registry 保存所有插件。Go 插件必须编译进二进制，所以新增插件 = 在本包加文件
// 并在 init 中 Register，没有目录扫描式的自动发现。
var registry []named

type named struct {
	name string
	p    Plugin
}

func Register(name string, p Plugin) {
	registry = append(registry, named{name: name, p: p})
}

func relativePath(filePath, skillDir string) string {
	rel, err := filepath.Rel(skillDir, filePath)
	if err != nil {
		return filePath
	}
	return rel
}

// ScanDirectory 递归遍历 skillDir，对每个文件跑所有插件。
// 返回值以绝对路径为键，只包含至少有一个命中的文件。
//
// 插件/文件级别的 panic 被隔离并转成一条警告：单个插件崩掉不影响整体扫描。
func ScanDirectory(skillDir string) (map[string][]Issue, error) {
	results := map[string][]Issue{}
	absSkill, err := filepath.Abs(skillDir)
	if err != nil {
		return nil, err
	}
	var files []string
	walkErr := filepath.WalkDir(absSkill, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(files)

	for _, fp := range files {
		raw, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		content := string(raw)
		var issues []Issue
		for _, n := range registry {
			issues = append(issues, runPlugin(n, absSkill, fp, content)...)
		}
		if len(issues) > 0 {
			results[fp] = issues
		}
	}
	return results, nil
}

func runPlugin(n named, skillDir, filePath, content string) (issues []Issue) {
	defer func() {
		if r := recover(); r != nil {
			warnf("[%s] crashed on %s: %v", n.name, filePath, r)
		}
	}()
	return n.p.Scan(skillDir, filePath, content)
}

var warnSink func(format string, args ...any)

// SetWarnSink 让上层注入日志实现，避免 plugin 包依赖具体日志库。
func SetWarnSink(f func(format string, args ...any)) { warnSink = f }

func warnf(format string, args ...any) {
	if warnSink != nil {
		warnSink(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func lineOf(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}
