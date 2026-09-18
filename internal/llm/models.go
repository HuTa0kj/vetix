package llm

// 结构化输出的数据模型。字段的 json tag 必须与提示词里内嵌的 Python 类定义
// 逐字一致——它们既是从模型回复里解码的目标，也是 json_schema 模式下发给模型的
// schema 来源。注意两条路径的字段名本来就不一致（插件侧是 line，行为分析侧是
// line_number），这是 Python 版的既有差异，不要统一。

type RiskFinding struct {
	Name        string `json:"name" jsonschema:"description=Risk name, a short noun phrase"`
	Description string `json:"description"`
	Severity    string `json:"severity" jsonschema:"enum=info,enum=low,enum=medium,enum=high,enum=critical"`
	Category    string `json:"category" jsonschema:"description=Risk classification"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
}

type BehavioralRiskItem struct {
	Category    string `json:"category" jsonschema:"description=Risk classification"`
	Severity    string `json:"severity" jsonschema:"enum=low,enum=medium,enum=high,enum=critical"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Name        string `json:"name" jsonschema:"description=Risk name, a short noun phrase"`
	Description string `json:"description"`
}
