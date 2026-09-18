// Package buildinfo 保存版本与工具名。放在 internal 下是为了让 report 与 runner
// 都能引用而不产生循环依赖。
package buildinfo

const (
	Version  = "2.0.0"
	ToolName = "vetix"
	ToolDesc = "Vetix — Automated scanning, identification, and assessment of SKILL security risks."
)
