package runner

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"

	"vetix/internal/audit"
)

func TestPrintSummary(t *testing.T) {
	results := []skillResult{
		{Name: "pop-calc", Hash: "abc123def4567890", Counts: map[string]int{"critical": 2, "high": 1}},
		{Name: "readme-generator", Hash: "1a2b3c4d5e6f7a8b", Cached: true},
		{Name: "broken-one", Failed: true},
	}
	var outBuf bytes.Buffer
	printSummary(&outBuf, results)
	outStr := outBuf.String()

	if !strings.Contains(outStr, "Scan Summary (3 skills)") {
		t.Errorf("header with count missing:\n%s", outStr)
	}
	// 有发现的行：按等级细分，含彩色计数与哈希前缀。
	if !strings.Contains(outStr, "2 critical") || !strings.Contains(outStr, "1 high") {
		t.Errorf("severity breakdown missing:\n%s", outStr)
	}
	if !strings.Contains(outStr, "abc123def4567890") {
		t.Errorf("hash column missing:\n%s", outStr)
	}
	// 缓存命中的行没有发现，应为 Clean 并标注 cached。
	if !strings.Contains(outStr, "Clean") || !strings.Contains(outStr, "(cached)") {
		t.Errorf("clean/cached markers missing:\n%s", outStr)
	}
	// 失败的行：哈希占位，名字仍然出现。
	if !strings.Contains(outStr, "Failed") || !strings.Contains(outStr, "-") {
		t.Errorf("failed row markers missing:\n%s", outStr)
	}
}

func TestPrintSummaryAlignment(t *testing.T) {
	results := []skillResult{
		{Name: "short", Hash: "abc123def4567890"},
		{Name: "a-very-long-skill-name", Hash: "1a2b3c4d5e6f7a8b"},
	}
	var buf bytes.Buffer
	printSummary(&buf, results)

	// 哈希是定宽 16 列：两行的哈希列必须起在同一列上，名字列补空格对齐。
	var offsets []int
	for _, line := range strings.Split(buf.String(), "\n") {
		if idx := strings.Index(line, "abc123def4567890"); idx >= 0 {
			offsets = append(offsets, idx)
		}
		if idx := strings.Index(line, "1a2b3c4d5e6f7a8b"); idx >= 0 {
			offsets = append(offsets, idx)
		}
	}
	if len(offsets) != 2 || offsets[0] != offsets[1] {
		t.Fatalf("hash columns must align, got offsets %v:\n%s", offsets, buf.String())
	}
}

func TestNewSkillResultHashPrefix(t *testing.T) {
	s := &audit.State{SkillName: "demo", DirectoryHash: strings.Repeat("a", 64)}
	res := newSkillResult(s, false)
	if res.Hash != strings.Repeat("a", 16) {
		t.Fatalf("hash must be the 16-char prefix, got %q", res.Hash)
	}
	if res.Name != "demo" {
		t.Fatalf("name = %q, want demo", res.Name)
	}
}

func TestSeverityCellCleanAndColors(t *testing.T) {
	clean := severityCell(skillResult{Name: "x"})
	if !strings.Contains(text.StripEscape(clean), "Clean") {
		t.Fatalf("clean cell must say Clean: %q", clean)
	}
	cached := severityCell(skillResult{Name: "n", Cached: true})
	if !strings.Contains(text.StripEscape(cached), "(cached)") {
		t.Fatalf("cached cell must be annotated: %q", cached)
	}
	failed := severityCell(skillResult{Name: "n", Failed: true})
	if !strings.Contains(text.StripEscape(failed), "Failed") {
		t.Fatalf("failed cell must say Failed: %q", failed)
	}
}
