package plugin

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

// 单行内 40+ 个连续水平空白（空格、制表符、NBSP、各类排版空格、全角空格）
// 且后面还跟着内容：这段内容会被推出终端可视宽度之外，复制粘贴时才会带入。
// 只算行内水平空白，换行归 consecutive_newlines 管；行尾的纯空白不算
var horizontalPaddingRe = regexp.MustCompile(`[\t \x{00A0}\x{2000}-\x{200A}\x{202F}\x{205F}\x{3000}]{40,}\S`)

type HorizontalPaddingCheckPlugin struct{}

func (HorizontalPaddingCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "horizontal_padding",
		Name:        "Horizontal Whitespace Padding",
		Description: "Detects runs of 40+ horizontal whitespace characters followed by content, pushing that content beyond the visible terminal viewport.",
	}
}

func (HorizontalPaddingCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	var issues []Issue
	rel := relativePath(filePath, skillDir)
	// 同一行多个 padding run 只报一条。
	seen := map[int]bool{}
	for _, m := range horizontalPaddingRe.FindAllStringIndex(content, -1) {
		line := lineOf(content, m[0])
		if seen[line] {
			continue
		}
		seen[line] = true
		// run 长度按字符数报告，不是字节数。
		issues = append(issues, Issue{
			Name:     "Horizontal whitespace padding",
			Severity: SeverityHigh,
			Category: CatObfuscation,
			Description: fmt.Sprintf("The line contains a run of %d horizontal whitespace characters followed by content, "+
				"which may hide text beyond the visible terminal viewport.", utf8.RuneCountInString(content[m[0]:m[1]])),
			FilePath:      rel,
			Line:          line,
			AuditRequired: false,
		})
	}
	return issues
}
