package report

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/audit"
)

// 报告里的名称、路径、描述都来自被扫描的 SKILL，属于攻击者可控文本。原样打到终端
// 时 ESC 序列会被终端解释：伪造报告内容、清屏，甚至用 OSC 52 改写用户剪贴板。
func TestRenderStripsControlSequences(t *testing.T) {
	evil := "clean\x1b[2Jforged\x1b]52;c;cGF3bmVk\x07tail\rgone"
	got := sanitize(evil)
	if strings.ContainsRune(got, 0x1b) {
		t.Fatalf("escape sequences must be removed: %q", got)
	}
	// 整段序列一起吃掉，不要把 "[2J" 这样的序列体留成可见乱码。
	if strings.Contains(got, "[2J") || strings.Contains(got, "]52;") {
		t.Fatalf("sequence body leaked: %q", got)
	}
	if !strings.Contains(got, "clean") || !strings.Contains(got, "tail") {
		t.Fatalf("visible text must survive: %q", got)
	}
	// 换行与制表符是报告分段排版的一部分，不能一起删。
	if sanitize("a\tb\nc") != "a\tb\nc" {
		t.Errorf("newline and tab must be preserved: %q", sanitize("a\tb\nc"))
	}
	// 普通文本（含中文）不该被动。
	if s := "反向 shell: /dev/tcp/1.2.3.4/4444"; sanitize(s) != s {
		t.Errorf("plain text must pass through unchanged: %q", sanitize(s))
	}
}

// 未闭合的 CSI/DCS 序列（被截断的模型输出）要吞到串尾，不能漏出半个序列。
func TestSanitizeHandlesTruncatedSequence(t *testing.T) {
	for _, in := range []string{"x\x1b[38;5;", "x\x1b]8;;http://evil", "x\x1b"} {
		if got := sanitize(in); strings.ContainsRune(got, 0x1b) {
			t.Errorf("%q: leaked %q", in, got)
		}
	}
}

// 端到端确认渲染路径确实过了 sanitize，而不是只测了函数本身。
//
// 报告自带的配色本来就是 ANSI 序列（标题、严重度、暗色元信息），所以先去掉自己的
// 调色板会产出的全部序列，再看残留——残留只可能来自被扫描的 SKILL。
func TestRenderOutputHasNoEscapesFromFindings(t *testing.T) {
	s := &audit.State{
		SkillDir:    "/tmp/s\x1b[2J",
		SkillName:   "s\x1b[31m",
		OutputDir:   "/tmp/out",
		Language:    "en",
		Stats:       audit.TreeStats{Files: 1},
		SingleSkill: true,
	}
	s.PluginsVerifyFindings = []audit.RiskFinding{{
		Name: "n\x1b[2J", Severity: "high", Category: "Obfuscation\x1b]52;c;cGF3bmVk\x07",
		FilePath: "SKILL.md", Line: 1, Description: "d\x1b[2J",
	}}

	out := captureStdout(t, func() { Render(s) })
	own := map[string]bool{"\x1b[0m": true, "\x1b[1m": true}
	for _, c := range append(paletteColors(),
		text.FgHiBlue, text.FgGreen, text.FgHiBlack, text.FgWhite) {
		code := strconv.Itoa(int(c))
		own["\x1b["+code+"m"] = true          // 单色
		own["\x1b["+code+";1m"] = true        // 色 + 粗体（严重度徽标）
		own["\x1b[1;"+code+"m"] = true        // 粗体在前（text.Colors 的组合顺序可能不同）
	}
	for seq := range own {
		out = strings.ReplaceAll(out, seq, "")
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("attacker-controlled escapes survived into the report:\n%q", out)
	}
	if strings.Contains(out, "[2J") || strings.Contains(out, "]52;") {
		t.Fatalf("injected sequence body is visible in the report:\n%q", out)
	}
	if !strings.Contains(out, "SKILL.md") || !strings.Contains(out, "Obfuscation") {
		t.Errorf("visible content missing:\n%s", out)
	}
}

// paletteColors 汇总渲染会用到的所有严重度前景色。
func paletteColors() []text.Color {
	colors := make([]text.Color, 0, len(severityColors)+1)
	for _, c := range severityColors {
		colors = append(colors, c)
	}
	return append(colors, sevColor("unknown"))
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
