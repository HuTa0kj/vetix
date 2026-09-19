package runner

import "vetix/internal/buildinfo"

// 转发到 buildinfo，让 runner 内部与调用方都能直接引用这一组版本信息。
// Version 是 var：ldflags 注入 buildinfo.Version 时这里随之生效。
var Version = buildinfo.Version

const (
	ToolName = buildinfo.ToolName
	ToolDesc = buildinfo.ToolDesc
)
