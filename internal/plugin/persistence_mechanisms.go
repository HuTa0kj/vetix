package plugin

import (
	"fmt"
	"regexp"
)

var persistenceRules = []struct {
	label   string
	pattern *regexp.Regexp
}{
	// cron 任务写入
	{"Cron job", regexp.MustCompile(`(?i)\bcrontab\b|/var/spool/cron\b|/etc/cron\.(?:d|daily|hourly|weekly|monthly)\b`)},
	// shell 启动文件写入
	{"Shell startup file write", regexp.MustCompile(`(?i)(?:>>|>|\btee\b|\bcp\b|\bmv\b|\binstall\b|\bopen\b|\bwritefile\b|write_text)[^\n]*\.(?:bashrc|zshrc|zprofile|zshenv|bash_profile|zlogin|bash_aliases|profile)\b`)},
	// systemd 注册
	{"Systemd persistence", regexp.MustCompile(`(?i)\bsystemctl\s+(?:--user\s+)?enable\b|\.config/systemd/user/|/etc/systemd/system/`)},
	// launchd 注册
	{"LaunchAgent or LaunchDaemon", regexp.MustCompile(`(?i)Launch(?:Agents|Daemons)|\blaunchctl\s+(?:load|bootstrap|enable)\b`)},
	// 自启动目录
	{"Autostart entry", regexp.MustCompile(`(?i)\.config/autostart/|shell:startup|Start Menu.*Startup|\bchkconfig\b|\bupdate-rc\.d\b|/etc/rc\.local\b|/etc/init\.d/`)},
	// Windows 计划任务
	{"Windows scheduled task", regexp.MustCompile(`(?i)\bschtasks\s+/create\b`)},
	// Windows Run 注册表键
	{"Windows Run registry key", regexp.MustCompile(`(?i)CurrentVersion\\(?:Run|RunOnce)\b`)},
}

type PersistenceMechanismsCheckPlugin struct{}

func (PersistenceMechanismsCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "persistence_mechanisms",
		Name:        "Persistence Mechanisms",
		Description: "Detects attempts to persist across sessions: cron jobs, shell startup file writes, systemd or launchd registration, autostart entries and Windows scheduled tasks or Run keys.",
	}
}

func (PersistenceMechanismsCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	// 同一条规则在同一行去重：一个文件里反复出现同类持久化动作只报一条。
	seen := map[string]bool{}
	for i, rule := range persistenceRules {
		for _, m := range rule.pattern.FindAllStringIndex(content, -1) {
			line := lineOf(content, m[0])
			key := fmt.Sprintf("%d:%d", i, line)
			if seen[key] {
				continue
			}
			seen[key] = true
			text := content[m[0]:m[1]]
			if runes := []rune(text); len(runes) > 80 {
				text = string(runes[:80]) + "..."
			}
			issues = append(issues, Issue{
				Name:          "Persistence mechanism",
				Severity:      SeverityHigh,
				Category:      CatPersistence,
				Description:   fmt.Sprintf("References %s (%s).", rule.label, text),
				FilePath:      rel,
				Line:          line,
				AuditRequired: true,
			})
		}
	}
	return issues
}
