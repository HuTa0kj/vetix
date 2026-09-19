// Package buildinfo 保存版本与工具名。放在 internal 下是为了让 report 与 runner
// 都能引用而不产生循环依赖。
package buildinfo

// Version 用 var 保存，是为了发布时能通过 -ldflags "-X vetix/internal/buildinfo.Version=<tag>" 注入真实版本号。
var Version = "2.0.0"

const (
	ToolName = "vetix"
	ToolDesc = "Vetix — Automated scanning, identification, and assessment of SKILL security risks."
)
