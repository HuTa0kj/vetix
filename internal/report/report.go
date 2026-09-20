package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/audit"
	"vetix/internal/buildinfo"
	"vetix/internal/plugin"
)

var severityColors = map[string]text.Color{
	"critical": text.FgHiRed,
	"high":     text.FgRed,
	"medium":   text.FgYellow,
	"low":      text.FgGreen,
	"info":     text.FgCyan,
}

func sevColor(s string) text.Color {
	if c, ok := severityColors[strings.ToLower(s)]; ok {
		return c
	}
	return text.FgWhite
}

// sanitize 清掉来自被扫描 SKILL 的控制序列。报告里的名称、路径、描述都是攻击者可控
// 的文本，原样打到终端时 ESC 序列会被终端解释：轻则 `\x1b[2J` 清屏、伪造报告内容，
// 重则 OSC 52 直接把用户的剪贴板改写掉。一个专门读恶意文本的工具不该在交付结果这一
// 步把注入丢给用户。
//
// 只处理终端渲染这一条路径；report.json 保持原样，因为 encoding/json 本来就会把控制
// 字符转义成 \u00XX。
func sanitize(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == esc:
			// 整段序列一起吃掉，别把序列体（如 "[2J"）留成可见的乱码文本。
			i += escapeSeqLen(s[i:])
		case r == '\n' || r == '\t':
			sb.WriteString(s[i : i+size])
			i += size
		case isControl(r):
			i += size
		default:
			sb.WriteString(s[i : i+size])
			i += size
		}
	}
	return sb.String()
}

const esc = '\x1b'

func isControl(r rune) bool {
	return r == esc || r == 0x7f || (r < 0x20 && r != '\n' && r != '\t')
}

// escapeSeqLen 返回以 ESC 开头的整段转义序列的长度，无法识别时按两字符处理。
func escapeSeqLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch s[1] {
	case '[':
		// CSI：参数字节之后跟一个 0x40-0x7e 的终止字节。
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case ']', 'P', 'X', '^', '_':
		// OSC / DCS / SOS / PM / APC：以 BEL 或 ST 结束，被截断时吞到串尾。
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == esc && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	default:
		return 2
	}
}

// TaskDir 是报告的落盘目录：<output-dir>/<skill-hash 前 16 位>。
func TaskDir(s *audit.State) string {
	if s.DirectoryHash != "" {
		return filepath.Join(s.OutputDir, s.DirectoryHash[:16])
	}
	return s.OutputDir
}

type document struct {
	Metadata metadata        `json:"metadata"`
	Summary  summary         `json:"summary"`
	Findings []audit.Finding `json:"findings"`
}

type metadata struct {
	Tool       string `json:"tool"`
	Version    string `json:"version"`
	DetectedAt string `json:"detected_at"`
	TaskID     string `json:"task_id"`
	ThreadID   string `json:"thread_id"`
	SkillName  string `json:"skill_name"`
	SkillDir   string `json:"skill_dir"`
	Language   string `json:"language"`
	OutputDir  string `json:"output_dir"`
	SkillHash  string `json:"skill_hash"`
	FileNumber int    `json:"file_number"`
	// ScanKey 是扫描引擎指纹（版本 + 插件集），读缓存时据此判定缓存是否失效。
	ScanKey string `json:"scan_key"`
}

type summary struct {
	PluginFindings     int `json:"plugin_findings"`
	BehavioralFindings int `json:"behavioral_findings"`
	TotalFindings      int `json:"total_findings"`
}

func SkillName(s *audit.State) string {
	if s.SkillName != "" {
		return s.SkillName
	}
	return filepath.Base(strings.TrimRight(s.SkillDir, string(filepath.Separator)))
}

func build(s *audit.State) document {
	findings := s.Findings()
	if findings == nil {
		findings = []audit.Finding{}
	}
	return document{
		Metadata: metadata{
			Tool:       buildinfo.ToolName,
			Version:    buildinfo.Version,
			DetectedAt: s.DetectedAt,
			TaskID:     s.TaskID,
			ThreadID:   s.TaskID,
			SkillName:  SkillName(s),
			SkillDir:   s.SkillDir,
			Language:   orDefault(s.Language, "en"),
			OutputDir:  TaskDir(s),
			SkillHash:  s.DirectoryHash,
			FileNumber: s.FileNumber(),
			ScanKey:    plugin.Fingerprint(),
		},
		Summary: summary{
			PluginFindings:     len(s.PluginsVerifyFindings),
			BehavioralFindings: len(s.LLMFindings),
			TotalFindings:      len(findings),
		},
		Findings: findings,
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// WriteJSON 落盘 report.json。saveOutput 为 false 或没有输出目录时直接跳过。
func WriteJSON(s *audit.State) (string, error) {
	if !s.SaveOutput || s.OutputDir == "" {
		return "", nil
	}
	dir := TaskDir(s)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "report.json")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(build(s)); err != nil {
		return "", err
	}
	return path, nil
}

// Render 把审计结果打印到终端。这不是 report.json 的来源：两者都从 State 生成。
// 所有来自 SKILL 的文本都先过 sanitize。
//
// 布局是列表式：左边距固定，正文按终端宽度折行，没有任何跨行对齐的结构。表格布局
// 依赖"每行可见宽度严格相等"，一旦某行被终端折行（宽度探测偏差、ambiguous-width
// 字符、窗口缩放）整张表就错位；列表式布局退化得非常体面——终端再窄也只是自然
// 折行，缩进和层级都还在。
func Render(s *audit.State) {
	width := widthBudget()
	name := sanitize(SkillName(s))

	fmt.Println()
	fmt.Println(text.FgHiBlue.Sprint("────────── SKILL Security Audit Report: " + name + " ──────────"))
	fmt.Println()

	printKV("Skill", name, width)
	printKV("Directory", sanitize(s.SkillDir), width)
	printKV("Files", strconv.Itoa(s.FileNumber()), width)
	printKV("Language", orDefault(s.Language, "en"), width)

	if line := summaryLine(s); line != "" {
		fmt.Println()
		fmt.Println(line)
	}

	printSection("Plugin Findings", len(s.PluginsVerifyFindings), width)
	if len(s.PluginsVerifyFindings) == 0 {
		fmt.Println(text.FgGreen.Sprint("No plugin findings."))
	}
	for i, f := range s.PluginsVerifyFindings {
		printFinding(i+1, f.Name, f.Severity, f.Category, f.FilePath, f.Line, f.Description, width, i > 0)
	}

	behavioral := behavioralCount(s)
	printSection("Behavioral Analysis Findings", behavioral, width)
	if behavioral == 0 {
		fmt.Println(text.FgGreen.Sprint("No behavioral findings."))
	}
	n := 0
	for _, f := range s.LLMFindings {
		if f == nil {
			continue
		}
		n++
		printFinding(n, f.Name, f.Severity, f.Category, f.FilePath, f.LineNumber, f.Description, width, n > 1)
	}

	fmt.Println()
	fmt.Println(text.FgHiBlue.Sprint("────────── End of Report ──────────"))
	fmt.Println()
}

// 缩进口径：finding 头行是「2 空格 + %2d. 」共 5 列，正文全部悬挂缩进到同一列。
const (
	contentIndent = 5
	kvIndent      = 13 // 基本信息值列：2 缩进 + 9 键宽 + 2 间隔
)

// printSection 打印小节标题：粗体标题 + 暗色计数 + 一条暗色横线。
func printSection(title string, n int, width int) {
	fmt.Println()
	fmt.Printf("  %s %s\n", text.Bold.Sprint(title), text.FgHiBlack.Sprintf("· %d", n))
	fmt.Println("  " + text.FgHiBlack.Sprint(strings.Repeat("─", ruleWidth(width))))
}

func ruleWidth(width int) int {
	if w := width - 4; w > 20 {
		return w
	}
	return 20
}

// printKV 打印基本信息的一行：暗色键 + 值。值太长时折行并悬挂缩进，深路径不会把
// 行顶出终端。
func printKV(key, val string, width int) {
	lines := wrapIndent(val, width, kvIndent)
	fmt.Printf("  %s  %s\n", text.FgHiBlack.Sprintf("%-9s", key), lines[0])
	for _, l := range lines[1:] {
		fmt.Println(l)
	}
}

// printFinding 打印一条发现：
//
//  1. ● CRITICAL · Remote Execution · SKILL.md:31
//     Base64-encoded curl payload piped to bash
//
//     Line 31, presented as a MacOS setup instruction, ...
//
// 标题与描述各自独立折行，互不影响；gap 为 true 时先空一行与上一条隔开。
func printFinding(num int, name, severity, category, file string, line int, desc string, width int, gap bool) {
	name, severity = sanitize(name), sanitize(severity)
	category, file, desc = sanitize(category), sanitize(file), sanitize(desc)

	color := sevColor(severity)
	chip := text.Colors{color, text.Bold}.Sprint(fmt.Sprintf("%-8s", strings.ToUpper(orDefault(severity, "unknown"))))

	if gap {
		fmt.Println()
	}
	head := fmt.Sprintf("  %s %s %s", fmt.Sprintf("%2d.", num), text.Colors{color}.Sprint("●"), chip)
	if meta := metaLine(category, file, line); meta != "" {
		head += " " + text.FgHiBlack.Sprint("· "+meta)
	}
	fmt.Println(head)

	for i, l := range wrapIndent(name, width, contentIndent) {
		if i == 0 {
			fmt.Println("     " + text.Bold.Sprint(l))
		} else {
			fmt.Println(l)
		}
	}

	if strings.TrimSpace(desc) != "" {
		fmt.Println()
		for i, l := range wrapIndent(desc, width, contentIndent) {
			if i == 0 {
				fmt.Println("     " + l)
			} else {
				fmt.Println(l)
			}
		}
	}
}

// metaLine 组合「Category · File:Line」，缺哪段就省哪段。
func metaLine(category, file string, line int) string {
	var parts []string
	if category = strings.TrimSpace(category); category != "" {
		parts = append(parts, category)
	}
	switch {
	case file != "" && line > 0:
		parts = append(parts, fmt.Sprintf("%s:%d", file, line))
	case file != "":
		parts = append(parts, file)
	}
	return strings.Join(parts, " · ")
}

// summaryLine 汇总两个来源的严重度分布：「4 findings · 2 critical · 1 high」。
func summaryLine(s *audit.State) string {
	counts := SeverityCounts(s)
	total := len(s.PluginsVerifyFindings) + behavioralCount(s)
	if total == 0 {
		return ""
	}
	label := "findings"
	if total == 1 {
		label = "finding"
	}
	parts := append([]string{fmt.Sprintf("%d %s", total, label)}, severityParts(counts)...)
	return "  " + strings.Join(parts, text.FgHiBlack.Sprint(" · "))
}

// severityParts 返回按等级细分的彩色计数段，零计数不出现。
func severityParts(counts map[string]int) []string {
	var parts []string
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		if n := counts[sev]; n > 0 {
			parts = append(parts, text.Colors{sevColor(sev)}.Sprint(fmt.Sprintf("%d %s", n, sev)))
		}
	}
	return parts
}

func behavioralCount(s *audit.State) int {
	n := 0
	for _, f := range s.LLMFindings {
		if f != nil {
			n++
		}
	}
	return n
}

// wrapIndent 把文本按宽度折行，续行缩进 indent 列；返回的首行不带缩进，由调用方
// 按自己的前缀拼接。描述里的换行是 LLM 输出的一部分，按原样分段。窄到放不下时
// 保底 16 列，宁可让终端折行也不把词切成一个字母一列。
func wrapIndent(s string, width, indent int) []string {
	body := width - indent - 2
	if body < 16 {
		body = 16
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			out = append(out, "")
			continue
		}
		// WrapSoft 会把每行 pad 到等宽（软折行为了原地重绘），复制出来带尾随空格，
		// 在这里统一剪掉——空行注释里"不带尾随空格"的口径因此对每一行都成立。
		for _, seg := range strings.Split(text.WrapSoft(para, body), "\n") {
			out = append(out, strings.TrimRight(seg, " "))
		}
	}
	for i := 1; i < len(out); i++ {
		// 空行不补缩进：打出来是纯换行，复制出来的文本也不带尾随空格。
		if out[i] != "" {
			out[i] = strings.Repeat(" ", indent) + out[i]
		}
	}
	return out
}

// Counts 供调用方打印摘要日志。
func Counts(s *audit.State) (int, int) {
	return len(s.PluginsVerifyFindings), len(s.LLMFindings)
}

// Sanitize 是 sanitize 的导出口径：批量扫描汇总里要打印的 skill 名来自被扫描
// 目录，与报告正文同属攻击者可控文本，出报告包的终端输出都必须过它。
func Sanitize(s string) string {
	return sanitize(s)
}

// SeverityCounts 统计两个来源合并后的按等级发现数，键即 severity 字段小写。
func SeverityCounts(s *audit.State) map[string]int {
	counts := map[string]int{}
	for _, f := range s.PluginsVerifyFindings {
		counts[strings.ToLower(f.Severity)]++
	}
	for _, f := range s.LLMFindings {
		if f != nil {
			counts[strings.ToLower(f.Severity)]++
		}
	}
	return counts
}

// SeverityBreakdown 把按等级计数渲染成「2 critical · 1 high」的彩色串，零计数
// 不出现；全零时返回空串。空串的判定交给调用方（如渲染成 Clean）。
func SeverityBreakdown(counts map[string]int) string {
	sep := text.FgHiBlack.Sprint(" · ")
	return strings.Join(severityParts(counts), sep)
}
