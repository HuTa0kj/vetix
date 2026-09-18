package plugin

// 插件注册表。Python 版按文件名排序扫描 vetix/plugins/*.py 自动发现，Go 必须
// 编译进二进制，所以这里显式登记；顺序与 Python 的 sorted(glob) 保持一致。
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
