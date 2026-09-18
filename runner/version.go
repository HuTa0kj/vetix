package runner

import "vetix/internal/buildinfo"

// 转发到 buildinfo，让 runner 内部与调用方都能直接引用 runner.Version 这一组常量。
const (
	Version  = buildinfo.Version
	ToolName = buildinfo.ToolName
	ToolDesc = buildinfo.ToolDesc
)
