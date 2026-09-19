package runner

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/report"
)

// confirmScan 列出发现的 skills 并请求确认。EOF、空输入与任何非肯定回答一律视为
// 拒绝：批量扫描是交互动作，输入不是终端时多半来自管道或定时任务，宁可静默取消
// 也不误扫几十个 skill。
func confirmScan(w io.Writer, r io.Reader, root string, skills []string) (bool, error) {
	fmt.Fprintf(w, "\n%s\n\n", text.FgHiBlue.Sprintf("────────── Discovered Skills (%d) under %s ──────────", len(skills), root))
	for i, dir := range skills {
		fmt.Fprintf(w, "  %s  %s\n", text.FgHiBlack.Sprintf("%3d.", i+1), text.Bold.Sprint(report.Sanitize(filepath.Base(dir))))
	}
	fmt.Fprintf(w, "\nScan all %d skills? [y/N] ", len(skills))

	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
