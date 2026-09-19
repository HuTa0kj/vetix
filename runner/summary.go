package runner

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/audit"
	"vetix/internal/report"
)

// skillResult 记录单个 skill 的扫描汇总。Hash 是目录哈希前 16 位，即
// <output-dir>/<hash>/report.json 的子目录名——汇总表只做索引，凭它直接拼出
// 报告落盘路径；任务 ID 反而做不到这一点，只能往回翻终端记录。
type skillResult struct {
	Name   string
	Hash   string
	Cached bool
	Failed bool
	Counts map[string]int
}

// newSkillResult 从扫描完成或缓存回读的 State 收集汇总行。Hash 为空说明没算出
// 目录哈希（通常是中途失败），汇总表里以 failed 占位。
func newSkillResult(s *audit.State, cached bool) skillResult {
	res := skillResult{
		Name:   report.SkillName(s),
		Cached: cached,
		Counts: report.SeverityCounts(s),
	}
	if s.DirectoryHash != "" {
		res.Hash = s.DirectoryHash[:16]
	}
	return res
}

// printSummary 渲染批量扫描的收尾汇总表。名字来自被扫描目录，先过 sanitize；
// 列式布局在此处是安全的：名称列之外的两列都是固定或短内容，不会把行顶出终端。
func printSummary(w io.Writer, results []skillResult) {
	nameWidth := len("Skill")
	for _, r := range results {
		if n := utf8.RuneCountInString(r.Name); n > nameWidth {
			nameWidth = n
		}
	}

	fmt.Fprintf(w, "\n%s\n\n", text.FgHiBlue.Sprintf("────────── Scan Summary (%d skills) ──────────", len(results)))
	for _, r := range results {
		pad := strings.Repeat(" ", nameWidth-utf8.RuneCountInString(r.Name))
		fmt.Fprintf(w, "  %s%s  %s  %s\n", r.Name, pad, severityCell(r), hashCell(r))
	}
	fmt.Fprintln(w)
}

// severityCell 渲染按等级细分的计数；零发现给绿色的 Clean，缓存命中加注，
// 提醒用户这一行的数字不是本次扫描跑出来的。
func severityCell(r skillResult) string {
	if r.Failed {
		return text.FgHiRed.Sprint("Failed")
	}
	breakdown := report.SeverityBreakdown(r.Counts)
	if breakdown == "" {
		breakdown = text.FgGreen.Sprint("Clean")
	}
	if r.Cached {
		breakdown += " " + text.FgHiBlack.Sprint("(cached)")
	}
	return breakdown
}

func hashCell(r skillResult) string {
	if r.Hash == "" {
		return text.FgHiBlack.Sprint("-")
	}
	return text.FgHiBlack.Sprint(r.Hash)
}
