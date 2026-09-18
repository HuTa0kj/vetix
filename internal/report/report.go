package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/audit"
	"vetix/internal/buildinfo"
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

// Render 把审计结果打印到终端。这不是 report.json 的来源：两者都从 State 生成，
// 但终端表格为了可读性做了换行与截断。
func Render(s *audit.State) {
	name := SkillName(s)
	pluginFindings := s.PluginsVerifyFindings
	llmFindings := s.LLMFindings

	fmt.Println()
	fmt.Println(text.FgHiBlue.Sprint("────────── SKILL Security Audit Report: " + name + " ──────────"))
	fmt.Println()

	info := table.NewWriter()
	info.SetStyle(table.StyleLight)
	info.SetTitle(text.Bold.Sprint(" Basic Info "))
	info.AppendRow(table.Row{text.Bold.Sprint("Skill"), name})
	info.AppendRow(table.Row{text.Bold.Sprint("Directory"), s.SkillDir})
	info.AppendRow(table.Row{text.Bold.Sprint("Files"), s.FileNumber()})
	info.AppendRow(table.Row{text.Bold.Sprint("Language"), orDefault(s.Language, "en")})
	fmt.Println(info.Render())
	fmt.Println()

	if len(pluginFindings) > 0 {
		t := newFindingsTable(" Plugin Findings ")
		for i, f := range pluginFindings {
			t.AppendRow(table.Row{
				i + 1,
				text.Colors{sevColor(f.Severity)}.Sprint(f.Severity),
				f.Name, f.Category, f.FilePath, f.Line, f.Description,
			})
		}
		fmt.Println(t.Render())
		fmt.Println()
	} else {
		fmt.Println(text.FgGreen.Sprint("No plugin findings."))
		fmt.Println()
	}

	if len(llmFindings) > 0 {
		t := newFindingsTable(" Behavioral Analysis Findings ")
		for i, f := range llmFindings {
			if f == nil {
				continue
			}
			t.AppendRow(table.Row{
				i + 1,
				text.Colors{sevColor(f.Severity)}.Sprint(f.Severity),
				f.Name, f.Category, f.FilePath, f.LineNumber, f.Description,
			})
		}
		fmt.Println(t.Render())
		fmt.Println()
	} else {
		fmt.Println(text.FgGreen.Sprint("No behavioral findings."))
		fmt.Println()
	}

	fmt.Println(text.FgHiBlue.Sprint("────────── End of Report ──────────"))
	fmt.Println()
}

func newFindingsTable(title string) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetTitle(text.Bold.Sprint(title))
	t.AppendHeader(table.Row{"#", "Severity", "Name", "Category", "File", "Line", "Description"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 3},
		{Number: 2, WidthMax: 10},
		{Number: 3, WidthMax: 28},
		{Number: 4, WidthMax: 20},
		{Number: 5, WidthMax: 30},
		{Number: 6, WidthMax: 6, Align: text.AlignRight},
		{Number: 7, WidthMax: 50},
	})
	return t
}

// Counts 供调用方打印摘要日志。
func Counts(s *audit.State) (int, int) {
	return len(s.PluginsVerifyFindings), len(s.LLMFindings)
}
