package audit

import (
	"encoding/json"
	"os"
	"path/filepath"

	"vetix/internal/pluginutils"
)

// report doc 与 report 包共用同一份 JSON 形状；这里只声明读取所需的子集。
type cachedReport struct {
	Metadata struct {
		TaskID     string `json:"task_id"`
		SkillDir   string `json:"skill_dir"`
		SkillName  string `json:"skill_name"`
		Language   string `json:"language"`
		DetectedAt string `json:"detected_at"`
		SkillHash  string `json:"skill_hash"`
		FileNumber int    `json:"file_number"`
	} `json:"metadata"`
	Findings []Finding `json:"findings"`
}

// LoadCachedReport 按 <output-dir>/<hash[:16]>/report.json 找已有报告。
// 与 Python 版一致：任何异常都退化成"没有缓存"，由调用方重新扫描。
func LoadCachedReport(skillDir, outputDir string) (*State, error) {
	hash, err := pluginutils.DirectoryHash(skillDir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(outputDir, hash[:16], "report.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var doc cachedReport
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil
	}

	s := &State{
		TaskID:        doc.Metadata.TaskID,
		SkillDir:      orStr(doc.Metadata.SkillDir, skillDir),
		SkillName:     doc.Metadata.SkillName,
		DirectoryHash: orStr(doc.Metadata.SkillHash, hash),
		OutputDir:     outputDir,
		SaveOutput:    false,
		DetectedAt:    doc.Metadata.DetectedAt,
		Language:      orStr(doc.Metadata.Language, "en"),
		// 缓存回读后不再触发单文件快速路径，文件数直接取记录值。
		SingleSkill: false,
		Stats:       TreeStats{Files: fileNumber(skillDir, doc.Metadata.FileNumber)},
	}
	for _, f := range doc.Findings {
		switch f.Source {
		case "plugin":
			s.PluginsVerifyFindings = append(s.PluginsVerifyFindings, RiskFinding{
				Name: f.Name, Description: f.Description, Severity: f.Severity,
				Category: f.Category, FilePath: f.FilePath, Line: f.Line,
			})
		default:
			s.LLMFindings = append(s.LLMFindings, &BehavioralRiskItem{
				Name: f.Name, Description: f.Description, Severity: f.Severity,
				Category: f.Category, FilePath: f.FilePath, LineNumber: f.Line,
			})
		}
	}
	return s, nil
}

func orStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func fileNumber(skillDir string, cached int) int {
	if cached > 0 {
		return cached
	}
	tree, err := pluginutils.Tree(skillDir)
	if err != nil {
		return 0
	}
	if isSingleSkillFile(skillDir) {
		return 1
	}
	_, files, _ := pluginutils.TreeStats(tree)
	return files
}
