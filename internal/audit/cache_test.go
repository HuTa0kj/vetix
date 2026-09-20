package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vetix/internal/plugin"
	"vetix/internal/pluginutils"
)

func writeCachedReport(t *testing.T, skillDir, outputDir, scanKey string) {
	t.Helper()
	hash, err := pluginutils.DirectoryHash(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(outputDir, hash[:16])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{
		"metadata": map[string]any{
			"task_id":     "task-1",
			"skill_name":  "demo",
			"skill_hash":  hash,
			"file_number": 1,
			"scan_key":    scanKey,
		},
		"findings": []any{},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCachedReportKeyMismatch(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()

	writeCachedReport(t, skillDir, outputDir, plugin.Fingerprint())
	cached, err := LoadCachedReport(skillDir, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if cached == nil {
		t.Fatal("report with the current scan_key must be a cache hit")
	}

	writeCachedReport(t, skillDir, outputDir, "0000000000000000")
	cached, err = LoadCachedReport(skillDir, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if cached != nil {
		t.Fatal("a mismatching scan_key must be treated as no cache")
	}
}

func TestLoadCachedReportLegacyFormat(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()

	// 旧格式报告没有 scan_key 字段，零值不匹配当前指纹，必须失效重扫。
	writeCachedReport(t, skillDir, outputDir, "")
	cached, err := LoadCachedReport(skillDir, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if cached != nil {
		t.Fatal("a report without scan_key must be treated as no cache")
	}
}
