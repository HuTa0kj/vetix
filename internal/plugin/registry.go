package plugin

// 插件注册表。Go 二进制没有运行时发现机制，插件必须显式登记；顺序即插件执行
// 顺序，保持按名字排序。
func init() {
	Register("base64_exec", Base64ExecPlugin{})
	Register("binary_file", BinaryFileCheckPlugin{})
	Register("consecutive_newlines", ConsecutiveNewlinesCheckPlugin{})
	Register("exceptional_file", ExceptionalFileCheckPlugin{})
	Register("large_file", LargeFileCheckPlugin{})
	Register("long_file", LongFileCheckPlugin{})
	Register("public_ip", PublicIPCheckPlugin{})
	Register("rare_file", RareFileCheckPlugin{})
	Register("reverse_shell", ReverseShellPlugin{})
}
