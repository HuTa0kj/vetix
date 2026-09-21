package plugin

import (
	"fmt"
	"regexp"
)

var credentialPathRules = []struct {
	label   string
	pattern *regexp.Regexp
}{
	{"SSH keys or SSH config directory", regexp.MustCompile(`\.ssh/`)},
	{"AWS credentials", regexp.MustCompile(`\.aws/(?:credentials|config)`)},
	{"GCP credentials", regexp.MustCompile(`\.config/gcloud/|application_default_credentials\.json`)},
	{"Azure credentials", regexp.MustCompile(`\.azure/`)},
	{"Kubernetes credentials", regexp.MustCompile(`\.kube/config|\bkubeconfig\b`)},
	{"Docker registry credentials", regexp.MustCompile(`\.docker/config\.json`)},
	{"Package registry credentials", regexp.MustCompile(`\.(?:npmrc|pypirc|netrc|git-credentials|htpasswd|pgpass)\b`)},
	{"System account files", regexp.MustCompile(`/etc/(?:passwd|shadow|sudoers)\b`)},
	// .env 后面必须跟分隔符或行尾：排除 .envrc、.env.example 这类同前缀文件。
	{"Environment file", regexp.MustCompile(`\.env(?:\.(?:local|production|development|dev|prod))?(?:$|['"\s)\],;:])`)},
	{"Credential-like file", regexp.MustCompile(`\b(?:password|passwd|credentials?|secrets?|tokens?|api_?keys?)\.(?:txt|json|ya?ml|env|pem|key)\b`)},
	{"Browser credential store", regexp.MustCompile(`(?:Chrome|Firefox|Safari|Edge)/\S*(?:Cookies|Login Data|logins\.json|key4\.db)`)},
}

type CredentialPathsCheckPlugin struct{}

func (CredentialPathsCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "credential_paths",
		Name:        "Credential File Paths",
		Description: "Detects references to local credential stores such as SSH keys, cloud provider configs, browser cookie databases and other secret files.",
	}
}

func (CredentialPathsCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	// 同一条规则在同一行去重：一个文件里反复列出同一类凭据路径只报一条。
	seen := map[string]bool{}
	for i, rule := range credentialPathRules {
		for _, m := range rule.pattern.FindAllStringIndex(content, -1) {
			line := lineOf(content, m[0])
			key := fmt.Sprintf("%d:%d", i, line)
			if seen[key] {
				continue
			}
			seen[key] = true
			text := content[m[0]:m[1]]
			// 按字符截断，避免把多字节 UTF-8 字符切成半个码点。
			if runes := []rune(text); len(runes) > 80 {
				text = string(runes[:80]) + "..."
			}
			issues = append(issues, Issue{
				Name:          "Credential file access",
				Severity:      SeverityHigh,
				Category:      CatSensitiveFileAccess,
				Description:   fmt.Sprintf("References %s (%s).", rule.label, text),
				FilePath:      rel,
				Line:          line,
				AuditRequired: true,
			})
		}
	}
	return issues
}
