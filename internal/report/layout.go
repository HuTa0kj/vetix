package report

import (
	"os"
	"strconv"
	"sync"

	"golang.org/x/term"
)

// 终端宽度探测。报告是列表式布局，没有跨行对齐的结构，宽度只用来决定文本在哪里
// 折行（见 report.go 的 wrapIndent）：探测准了折行就贴着右边距，探测不到也只是
// 折得早一点或晚一点，不会出现表格那种整行错位。
//
// 只探测一次：终端宽度在一次扫描里不会变，不值得每段文本都去 ioctl。
const (
	// defaultWidth 用于取不到终端宽度时（CI、重定向）。120 比大多数默认终端窗口略窄。
	defaultWidth = 120
	minWidth     = 72
	maxWidth     = 200
)

// widthBudget 返回文本折行可用的总列数。
func widthBudget() int {
	return onceWidth()
}

// WrapIndent 按终端宽度折行文本，续行悬挂缩进 indent 列，首行不带缩进。
// 导出给 report 自身之外的终端输出（如 runner 的 -pl）复用同一套折行口径，
// 免得每处自算宽度、阈值漂移。
func WrapIndent(s string, indent int) []string {
	return wrapIndent(s, widthBudget(), indent)
}

var onceWidth = sync.OnceValue(func() int { return clampWidth(detectWidth()) })

func clampWidth(w int) int {
	if w < minWidth {
		return minWidth
	}
	if w > maxWidth {
		return maxWidth
	}
	return w
}

func detectWidth() int {
	for _, probe := range []func() (int, bool){
		stdoutWidth,
		controllingTerminalWidth,
		columnsEnv,
	} {
		if w, ok := probe(); ok {
			return w
		}
	}
	return defaultWidth
}

func stdoutWidth() (int, bool) {
	return sizeOf(os.Stdout)
}

// controllingTerminalWidth 覆盖 stdout 不是终端的情况：vetix 会把 stdout 换成管道来
// 过滤 langsmith 的调试输出，这时 stdout 探测不到宽度，但用户确实坐在终端前面。
func controllingTerminalWidth() (int, bool) {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return 0, false
	}
	defer f.Close()
	return sizeOf(f)
}

func sizeOf(f *os.File) (int, bool) {
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}

func columnsEnv() (int, bool) {
	w, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}
