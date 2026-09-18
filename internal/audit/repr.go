package audit

import (
	"strings"

	"vetix/internal/plugin"
	"vetix/internal/pytext"
)

// severityMember 给出枚举成员名，用于构造 <Severity.CRITICAL: 'critical'> 形式。
func severityMember(s plugin.Severity) string {
	switch s {
	case plugin.SeverityInfo:
		return "INFO"
	case plugin.SeverityLow:
		return "LOW"
	case plugin.SeverityMedium:
		return "MEDIUM"
	case plugin.SeverityHigh:
		return "HIGH"
	case plugin.SeverityCritical:
		return "CRITICAL"
	}
	return strings.ToUpper(string(s))
}

// IssueRepr 按 dataclass 风格渲染一条命中：
//
//	Issue(name='x', description='y', severity=<Severity.CRITICAL: 'critical'>,
//	      file_path='p', category='Remote Execution', line=1, audit_required=True)
//
// 字段顺序不能改，因为这段文本会作为 "Hit list" 直接进入复核提示词。
func IssueRepr(i plugin.Issue) string {
	var sb strings.Builder
	sb.WriteString("Issue(name=")
	sb.WriteString(pytext.Quote(i.Name))
	sb.WriteString(", description=")
	sb.WriteString(pytext.Quote(i.Description))
	sb.WriteString(", severity=")
	sb.WriteString(pytext.EnumValue("Severity", severityMember(i.Severity), string(i.Severity)))
	sb.WriteString(", file_path=")
	sb.WriteString(pytext.Quote(i.FilePath))
	sb.WriteString(", category=")
	sb.WriteString(pytext.Quote(i.Category))
	sb.WriteString(", line=")
	sb.WriteString(pytext.Value(i.Line))
	sb.WriteString(", audit_required=")
	sb.WriteString(pytext.Value(i.AuditRequired))
	sb.WriteString(")")
	return sb.String()
}

// IssueListRepr 把命中列表渲染成 [Issue(...), Issue(...)] 的形式。
func IssueListRepr(issues []plugin.Issue) string {
	parts := make([]string, 0, len(issues))
	for _, i := range issues {
		parts = append(parts, IssueRepr(i))
	}
	return pytext.List(parts)
}
