package plugin

// 插件注册表。Go 二进制没有运行时发现机制，插件必须显式登记；顺序即插件执行
// 顺序，保持按 ID 排序。元数据（ID / Name / Description）由各插件文件里的
// Plugin.Meta() 自己声明，这里只负责登记，不保存任何插件的描述。
func init() {
	Register(Base64ExecPlugin{})
	Register(BinaryFileCheckPlugin{})
	Register(ConsecutiveNewlinesCheckPlugin{})
	Register(CredentialPathsCheckPlugin{})
	Register(ExceptionalFileCheckPlugin{})
	Register(HorizontalPaddingCheckPlugin{})
	Register(LargeFileCheckPlugin{})
	Register(LongFileCheckPlugin{})
	Register(PersistenceMechanismsCheckPlugin{})
	Register(PublicIPCheckPlugin{})
	Register(RareFileCheckPlugin{})
	Register(ReflectiveCallCheckPlugin{})
	Register(RemoteScriptExecCheckPlugin{})
	Register(ReverseShellPlugin{})
	Register(TyposquattingCheckPlugin{})
	Register(UnicodeConfusablesCheckPlugin{})
}
