package runner

import "vetix/internal/buildinfo"

// 转发到 buildinfo，保持 cmd/runner 层的引用风格与 GoTStarter 一致。
const (
	Version  = buildinfo.Version
	ToolName = buildinfo.ToolName
	ToolDesc = buildinfo.ToolDesc
)
