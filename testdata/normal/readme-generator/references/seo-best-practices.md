# SEO / GEO 最佳实践

## GitHub Repository Description（≤ 160 字符）

这是 Google 搜索结果里显示的摘要，也是 AI（ChatGPT/Perplexity/Gemini）引用项目时的描述来源。

**格式模板：**
```
[动词短语：功能描述] — [技术关键词1], [技术关键词2]. [可选：一个差异化说明]
```

**示例（好）：**
```
Generate beautifully typeset travel guidebook PDFs with parallel AI research agents and Playwright export
AI agent skill for creating professional travel itinerary books from natural language trip descriptions
```

**示例（差）：**
```
🚀 Amazing travel tool that makes your trips AWESOME!!
一个很强大的旅行工具
```

规则：
1. 英文优先（国际受众更广）
2. 动词开头（Generate/Build/Create/Convert/Automate）
3. 包含核心技术名词（作为 SEO 关键词）
4. 不用 emoji，不用感叹号，不说「very good」「amazing」
5. 描述「做什么」而不是「有多好」

---

## GitHub Topics（7-10 个）

Topics 直接影响 GitHub 搜索排名和 Explore 推荐。

**分类策略：**

| 类别 | 数量 | 示例 |
|------|------|------|
| 技术栈 | 2-3 | `nodejs`, `python`, `typescript` |
| 框架/工具 | 2-3 | `playwright`, `claude`, `react` |
| 应用领域 | 2-3 | `travel`, `pdf-generation`, `automation` |
| 项目类型 | 1-2 | `agent-skill`, `cli-tool`, `library` |

**避免：**
- 太宽泛：`software`, `tool`, `project`, `github`
- 重复信息：如果 description 里有 `nodejs`，topics 里可以省略
- 超过 10 个：质量比数量重要

---

## README H1 标题 SEO

H1 是爬虫权重最高的信号。

```markdown
# Travel Guidebook
```

规则：
- 必须是项目的正式名称（和仓库名一致或接近）
- 不在 H1 里塞关键词（Google 会降权）
- H1 之后的 blockquote 或 **bold** 标语可以包含自然关键词

---

## GEO（生成式引擎优化）

AI 搜索引擎（ChatGPT/Perplexity/Claude）引用 README 时，优先选择：

1. **结构清晰**：有明确的 H2 章节标题
2. **数据具体**：有具体数字（「5 分钟内生成」「支持 30+ 页」）
3. **代码示例**：有可执行的命令示例
4. **对比说明**：说明与其他方案的区别（「不是模板填充，而是...」）

在 README 开头的「这是什么」章节，用 1-2 句话精确描述项目，这段话很可能被 AI 直接引用。
