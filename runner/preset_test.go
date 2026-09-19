package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePreset(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}
	// 路径必须经 filepath.Join 拼出，不硬编码分隔符，Windows 才能解析对。
	cases := map[string]string{
		"claude-code": filepath.Join(home, ".claude", "skills"),
		"codex":       filepath.Join(home, ".codex", "skills"),
	}
	for name, want := range cases {
		got, err := resolvePreset(name)
		if err != nil {
			t.Fatalf("resolvePreset(%s): %v", name, err)
		}
		if got != want {
			t.Errorf("resolvePreset(%s) = %q, want %q", name, got, want)
		}
	}
}

func TestResolvePresetUnknown(t *testing.T) {
	_, err := resolvePreset("no-such-tool")
	if err == nil {
		t.Fatal("unknown preset must fail")
	}
	// 报错里列出可用预设，用户才知道能填什么。
	for _, name := range presetNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error must list available preset %s: %v", name, err)
		}
	}
}
