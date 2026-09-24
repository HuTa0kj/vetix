package runner

import (
	"bytes"
	"strings"
	"testing"

	"vetix/internal/plugin"
)

// -pl 的输出直接面对用户：头部计数、每个注册插件都要出现，且折行不得丢字。
func TestPrintPlugins(t *testing.T) {
	var buf bytes.Buffer
	printPlugins(&buf)
	out := buf.String()

	if !strings.Contains(out, "Built-in Plugins (16)") {
		t.Errorf("header with count missing:\n%s", out)
	}
	for _, m := range plugin.List() {
		if !strings.Contains(out, m.ID) || !strings.Contains(out, m.Name) {
			t.Errorf("plugin %s / %q missing from the listing", m.ID, m.Name)
		}
	}
	// 折行会把同一句话拆到多行，剥掉折行与色码后核对词序仍在。
	collapsed := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"exfiltration targets.", "actually read."} {
		if !strings.Contains(collapsed, want) {
			t.Errorf("wrapped description must not drop words: %q", want)
		}
	}
	// WrapSoft 会把每行 pad 到定宽，行尾空格必须在输出前剪掉。
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimRight(line, " \t") != line && line != "" {
			t.Errorf("line must not end with padding spaces: %q", line)
		}
	}
}
