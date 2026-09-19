---
name: readme-generator
description: 为 GitHub 项目生成作品集级 README.md。适用于「帮我写 README」「生成 README」「优化 README」「README 最佳实践」「项目首页」「开源说明」「README 信息图」「README 封面」「用 Codex Image Gen / gpt-image-2 生成 README 图片」等请求。输出包括克制的 README 叙事、最多两张高质量视觉资产、压缩后的图片、MIT 许可证、GitHub Description 和 Topics 推荐、推荐星级，以及可选 gh CLI 更新建议。重点是帮项目讲清自己的故事，并基于项目类型判断视觉强度，避免套模板、堆信息和过度设计。
---

# GitHub README Generator

README 是项目的第一张作品集页面。它不是说明书的目录，也不是功能清单的容器。它要在很短时间内回答三件事：

1. 这是什么。
2. 为什么值得看。
3. 我怎么开始使用。

本 Skill 的目标是生成 **100 分 README 作品**：清楚、有审美、克制、可信，能让项目像一个完整作品一样被理解。

---

## 第一性原理

README 是信任入口和路径入口，不是完整文档。它应该帮助第一次打开仓库的人做出快速判断：

1. 这个项目解决什么问题。
2. 它适不适合我。
3. 我是否能马上运行、安装或继续了解。

1w star 以上开源项目通常不是靠信息量取胜，而是靠清晰的首屏、直接的上手路径、可信的文档入口和克制的社区信息取胜。图片、徽章、作者信息和设计理念都只是辅助；一旦它们拖慢理解，就是噪音。

本 Skill 的核心取舍：README 先讲清项目，再做美化；视觉服务理解，不替代理解。

---

## 作品标准

用下面的评分表约束所有输出：

| 维度 | 分值 | 判断标准 |
|------|------|----------|
| 15 秒理解 | 25 | 首屏能看懂项目名、价值、适用对象 |
| 项目故事 | 20 | 不是堆功能，而是讲清背景、动机和结果 |
| 视觉表达 | 20 | 图片像作品，不像小字流程截图 |
| 快速开始 | 15 | 安装和使用路径短、明确、可复制 |
| 可信产物 | 10 | 展示真实输出、能力边界或结果 |
| 克制降噪 | 10 | 去掉重复、口号、过度解释和装饰 |

低于 90 分的 README 不交付；先删噪音、放大重点、重排叙事。

### 高星项目基线

默认向高星开源项目学习这些结构：

- 项目名 + 一句话价值主张。
- 少量必要 badge，不堆状态装饰。
- 快速开始或文档入口靠前。
- 示例只在能降低上手成本时出现。
- 贡献、社区、安全、许可证简洁清楚。
- UI / 产品项目可放截图；库、SDK、基础设施项目少图或无图。

不要把 README 写成设计宣言、完整说明书、功能墙、社交名片或内部工作流报告。

---

## 设计原则

### 必须坚持

- H1 必须是项目正式名称，紧跟一句价值主张。
- README 开头先讲项目价值，再放安装细节。
- 图片只表达一个重点，但应该承担项目名片功能：让读者一眼看到项目名和定位。
- 默认最多两张图片：一张封面，一张核心能力或结果图。
- GitHub 会缩小图片显示，图片里的主文案必须按海报字号设计。
- 对功能的描述要具体，但不夸张；能用结果说明就不要自夸。
- 对作者和许可证保持简洁，不做社交名片堆砌。

### 必须避免

- ASCII 艺术标题。
- emoji 装饰标题或作者表格。
- 大段“我们很专业”的空话。
- 6 个以上小卡片堆在一张图里。
- 流程图里塞满阶段、命令和小字说明。
- 把第三张流程图当作默认产物；工作方式通常用正文讲更清楚。
- 把 Image Gen 当作长文排版工具。
- 把 README 写成完整产品手册；详细文档应放到 `docs/`。

---

## 总体流程

```
Phase 0    项目阅读和模式识别
Phase 1    项目故事提炼
Phase 2    视觉生成方式选择
Phase 3    作品级视觉资产生成
Phase 4    README 组装
Phase 5    GitHub 元信息建议
Phase 6    验证和交付
```

---

## Phase 0: 项目阅读和模式识别

先读取项目，而不是直接写模板。

检查：

```bash
ls
find . -maxdepth 2 -type f | sed 's#^\./##' | sort | head -80
```

优先读取：

- `README.md`
- `package.json` / `pyproject.toml` / `go.mod` / `Cargo.toml`
- `docs/`
- 主要入口文件
- 示例、截图、演示文件

判断场景和视觉预算：

| 场景 | 判断方式 | 策略 | 图片上限 |
|------|----------|------|----------|
| 新建 README | 没有 README，或 README 很短 | 完整生成，但保持短路径 | 1-2 |
| 升级 README | 已有 README，有有效内容 | 保留独特内容，重写结构和首屏 | 0-2 |
| 作品集强化 | 用户强调审美、故事、展示 | 优先做叙事和封面表现 | 2 |
| 纯文档模式 | SDK、库、后端工具、基础设施 | 少图，重安装、API、文档入口 | 0-1 |
| UI / 产品展示 | 有界面、截图、demo、视觉结果 | 用真实结果或封面辅助理解 | 1-2 |

升级现有 README 时，不要删除用户已有的关键内容。先提取可保留内容，再重排。

### 降噪审查

写 README 前先标记哪些内容应外移或删除：

| 内容 | 默认处理 |
|------|----------|
| 设计评分表、工作原则、内部方法论 | 放在 `SKILL.md` 或 `docs/`，不进 README |
| 生成文件树、模板变量、脚本细节 | 只在用户需要开发文档时保留 |
| 第三张流程图 | 删除，改成 2-4 行正文或简短列表 |
| 过多社交链接 | 只留 GitHub / 主页等核心入口 |
| 太泛的功能列表 | 合并成 3 个结果导向能力 |
| 安装前的长背景 | 缩短，快速开始提前 |

---

## Phase 1: 项目故事提炼

采集或推断 7 个字段：

| 字段 | 说明 |
|------|------|
| `project_name` | 项目正式名称 |
| `tagline` | 一句话价值主张，短、有判断 |
| `origin` | 项目出现的背景：为什么需要它 |
| `audience` | 谁会用它 |
| `promise` | 它帮用户得到什么结果 |
| `proof` | 真实能力、截图、输出、示例、指标 |
| `start` | 最短上手路径 |

不要问太多问题。能从项目里推断就直接推断；只有影响叙事准确性时才问用户。

### 推荐叙事结构

```
项目名
一句话价值主张
视觉封面（按项目类型决定是否需要）

这是什么
为什么需要它
你会得到什么
快速开始
示例或输出
工作方式（正文，不默认配图）
安装
许可证
作者
```

如果项目偏工具、库或基础设施，把“快速开始”提前到“为什么需要它”之后。README 的顺序要服务读者行动，不服务模板完整性。

---

## Phase 2: 视觉生成方式选择

README 图片有两类：

1. **作品封面**：传达气质、主题、记忆点。
2. **结构说明图**：传达步骤、能力、对比、流程。

根据用户意图选择模式：

| 模式 | 适用场景 | 图片策略 |
|------|----------|----------|
| `portfolio` | 默认推荐，适合需要展示完整作品感的项目 | 1 张封面 + 1 张核心能力图 |
| `clean-doc` | SDK、库、后端工具、严肃基础设施 | 0-1 张图，优先快速开始和示例 |
| `visual-story` | AI 工具、设计工具、独立产品、作品展示 | 最多 2 张图，Codex Image Gen 负责记忆点 |
| `structured` | 用户明确要信息图、对比图、流程图 | 1-2 张 HTML/CSS 海报，保证文字准确 |

默认先判断项目类型，不要强行套 `portfolio`。视觉资产生成优先级固定为：

1. **Codex Image Gen Skill**：优先用 Codex 内置图片生成能力生成视觉资产。
2. **轻量化**：README 展示图优先转成 WebP，再写入 README 引用路径。
3. **HTML to PNG fallback**：只有当 Image Gen 不可用、输出不符合要求、或用户明确要求结构化精确文字时，才退化到 HTML/CSS 模板截图。

不要把 HTML 截图当默认路径。它是可靠兜底，不是首选视觉方案。

---

## Phase 3: 作品级视觉资产生成

默认输出使用两个稳定文件名：

```
assets/banner.webp
assets/features.webp
```

这样 README 引用路径稳定，不管图片来自 Codex Image Gen 还是 HTML 截图兜底。

### 图片职责

| 图片 | 目标 | 推荐方式 |
|------|------|----------|
| `banner.webp` | 项目名片，建立项目名、定位和记忆点 | Codex Image Gen 优先 |
| `features.webp` | 核心能力、结果或必要流程的传播图 | Codex Image Gen 优先；文字精确时 HTML 兜底 |

不要默认生成 `workflow.png` 或 `workflow.webp`。如果用户明确要求流程图，把流程内容合并进 `features.webp` 或放到正文，不新增第三张图。

### 默认视觉风格

所有 README 视觉资产默认采用同一套风格：

```text
黑底、极简、电影打光、高对比、大留白、低亮度、白/灰/暖金三色、高级杂志封面感。
画面质感：极深黑背景 #050505，纸张颗粒，浅景深，体积雾，细窄轮廓光，局部金属质感。
质量目标：出自 1w star 设计师水准作品。
```

设计约束：

- 背景以 `#050505` 深黑为主。
- 色彩只使用白、灰、暖金；不要引入彩虹渐变、紫蓝霓虹或高饱和装饰色。
- 使用大留白和局部光，而不是堆元素。
- 图片应该有少量高价值文字，承担传播和定位，不做长说明。
- 文字越少越强：每张图只保留读者离开 README 后仍该记住的信息。
- 不放段落、命令、表格、版本号堆叠和密集说明。

### 项目名片文字策略

默认使用「项目名片型图片」，不是纯氛围图。

| 图片 | 推荐文字 | 上限 |
|------|----------|------|
| `banner.webp` | 项目名 + 一句话定位 + 1-3 个短标签 | 18 个英文词或 28 个中文字 |
| `features.webp` | 2-3 个结果短语，必要时加一个短标题 | 每个短语 2-5 个词 |

好文字应该像封面标题，不像说明书：

- 项目名必须清楚，优先放在 `banner.webp`。
- 定位句说结果，不说口号，例如 “Portfolio-grade README design for open source projects”。
- 标签只放搜索和记忆价值最高的词，例如 `Story`、`Visual`、`Signal`。
- 中文可以用，但要少；英文项目名、短英文标签通常更稳。
- 如果 Image Gen 把文字写错，重试一次；仍不准确时，保留 Image Gen 视觉底图，用 HTML/CSS fallback 承载精确文字。
- 不为了“全程 AI 生成”牺牲项目名和定位的准确性。

### 字号底线

按 1920×1080 设计时：

| 元素 | 最小字号 |
|------|----------|
| 主标题 | 92px |
| 中文主标题 | 80px |
| 大卡标题 | 48px |
| 正文说明 | 28px |
| 辅助标签 | 22px |
| 页脚 | 20px |

不要使用 18px 以下文字。GitHub 缩放后会不可读。

Codex Image Gen 产物只要求保持 16:9 和足够清晰，不要为了凑 `1920×1080` 而把好图强行重采样。README 展示图优先保存为 WebP；`1920×1080` 是 HTML to PNG fallback 的模板尺寸。

### HTML to PNG fallback

仅在 Image Gen 不可用、用户要求精确结构化文字、或 Image Gen 输出无法通过检查时使用：

```bash
node scripts/gen_infographic.mjs /tmp/readme-banner.html assets/banner.png 1920 1080
node scripts/gen_infographic.mjs /tmp/readme-features.html assets/features.png 1920 1080
node scripts/convert_webp_assets.mjs assets/banner.png assets/banner.webp assets/features.png assets/features.webp
```

模板来自：

```
templates/banner.html
templates/features.html
```

模板变量：

```
{{PROJECT_NAME}}
{{TAGLINE}}
{{PRIMARY_COLOR}}
{{CATEGORY}}
{{PLATFORM}}
{{LANGUAGE}}
{{VERSION_INFO}}
{{TECH_CARDS}}
{{FEATURE_CARDS}}
{{FEATURE_COUNT}}
```

卡片结构：

- `{{TECH_CARDS}}` 用在封面右侧，建议 2-3 条，只放短标签和一句说明。
- `{{FEATURE_CARDS}}` 只放 2-3 张大卡，第一张可加 `featured`。
- 如果需要表达工作流程，用 `features.webp` 的 3 个结果阶段承载，不新增第三张图。

### Codex Image Gen / gpt-image-2 方式

默认优先调用 Codex 自带图片生成能力，不要在项目里临时硬编码 API 脚本。

适合 Image Gen 的内容：

- README 封面。
- 产品氛围图。
- 作品集视觉。
- 抽象概念图。
- 用现有图片做风格延展。

不适合 Image Gen 的内容：

- 精确流程图。
- 大量中文文字或多段说明。
- 命令、版本、表格。
- 必须逐字准确的长文或 UI 图。

推荐 prompt 结构：

```text
Use case: productivity-visual
Asset type: GitHub README visual asset, 16:9
Project: <project_name>
Story: <origin + promise>
Visual direction: black background, minimalist, cinematic lighting, high contrast, large negative space, low brightness, premium magazine cover
Texture and lighting: #050505 deep black background, subtle paper grain, shallow depth of field, volumetric haze, thin rim light, selective metallic highlights
Palette: white, gray, warm gold only
Composition: one strong visual idea, restrained, spacious, no dense UI
Quality bar: 10k-star designer portfolio quality
Text policy: include only high-value name-card text. Use the exact project name, one short positioning line, and optional 1-3 short labels. No paragraphs, commands, tables, tiny captions, or decorative text.
Avoid: emoji, clutter, fake interface text, tiny labels, generic gradients, bright neon, overdesigned dashboards
```

如果 Image Gen 不支持、无法调用、或输出出现错误文字/风格偏差，先用一个更短的文字 prompt 重试一次；如果仍不准，再退回 HTML to PNG 兜底流程承载精确文字。

### 图片轻量化

README 展示图片默认使用 WebP。WebP 通常比 PNG 更适合 README 加载；PNG 只作为 HTML 截图兜底的中间产物、透明图或需要无损保存时使用。

默认命令：

```bash
npm run webp
```

如果项目没有 WebP 转换脚本，优先使用 `cwebp`、`sips` 或可用的轻量工具。不要为了转换引入重型依赖。无法转换时再保留 PNG，并说明原因。

### 视觉检查

每张图生成后检查：

- 缩小到 GitHub README 显示宽度后仍能读。
- 一张图只讲一个重点。
- 没有小字堆叠。
- 项目名和定位句一眼可见。
- 图中文字没有拼写错误、乱码或伪文字。
- 没有过度装饰。
- 视觉风格和项目故事一致。
- Image Gen 图没有错误文字或多余水印。
- 压缩后图片仍然清晰，没有明显色带、噪点破坏或文字糊边。

---

## Phase 4: README 组装

默认结构：

```markdown
<div align="center">

# 项目名

**一句话价值主张**

<img src="assets/banner.webp" alt="[项目名] — [价值主张]" width="100%">

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

</div>

---

## 这是什么

[2-3 句话讲清项目、背景、结果。]

## 为什么需要它

[讲真实问题，不列反模式清单。]

## 你会得到什么

<img src="assets/features.webp" alt="[3 个核心能力]" width="100%">

## 工作方式

[2-4 行讲清工作方式；不要默认再放第三张图。]

## 快速开始

[最短可执行路径。]

## 安装

[依赖和安装命令。]

## 许可证

[MIT](./LICENSE)

## 关于作者

[简洁作者信息。]
```

规则：

- 不要超过 3 个 badge，除非项目确实需要状态标识。
- 不要在图片后重复同样的功能列表。
- 不要把“设计原则”“项目结构”“生成内容”都塞进 README；只保留对读者有用的部分。
- 不要默认放第三张 workflow 图；两张图已经足够承载作品感和核心信息。
- 如果有详细说明，放到 `docs/`，README 只做入口。

---

## Phase 5: GitHub 元信息推荐

Repository Description 和 Topics 是 GitHub 搜索、Trending 关联、AI 搜索引用和人工快速判断的入口。它们不是 README 内容，不要写进 `README.md`；只在最终回复里作为独立建议输出。

### First-principles 目标

- **Description** 回答：这个项目为谁创造什么结果。
- **Topics** 回答：这个项目应该被放进哪些搜索桶。
- **推荐数量** 由信号质量决定，不追求多；高质量 7-9 个通常优于铺满 20 个。
- **Trending / SEO / GEO** 三者平衡：既要覆盖热门搜索词，也要保持项目事实准确，方便 AI 引用。

### Description 生成规则

生成 3 个候选，并逐个打分：

```text
[Verb] [specific outcome] for [audience] — [1-3 precise keywords]
```

硬约束：

- 英文优先，≤ 160 字符。
- 动词开头：Design / Generate / Build / Create / Convert / Automate。
- 包含项目最核心的 2-3 个关键词。
- 不用 emoji、感叹号、营销形容词。
- 不写无法从项目事实支撑的能力。

评分：

| 维度 | 分值 | 判断 |
|------|------|------|
| 准确性 | 40 | 是否忠于项目真实能力 |
| 搜索价值 | 25 | 是否覆盖 GitHub / Google 常搜词 |
| AI 可引用性 | 20 | 是否能被 ChatGPT / Claude / Perplexity 直接理解 |
| 克制程度 | 15 | 是否没有夸张和废话 |

星级：

| 分数 | 星级 |
|------|------|
| 90-100 | 5 星，推荐使用 |
| 80-89 | 4 星，可用但可再精简 |
| 70-79 | 3 星，只适合备选 |
| < 70 | 不推荐 |

### Topics 生成规则

先建立候选池，再筛选最终推荐。

候选来源：

- 项目类型：`agent-skill`, `cli-tool`, `developer-tools`, `documentation`
- 核心技术：`nodejs`, `playwright`, `codex`, `image-generation`
- 应用领域：`readme`, `github-readme`, `open-source`, `portfolio`
- 当前趋势：AI agent、image generation、developer tooling、documentation automation
- 项目实际文件和 README 叙事中出现的关键词

筛选规则：

- 推荐 7-9 个，不超过 10 个，除非项目确实横跨多个明确领域。
- Topic 必须小写，只用字母、数字和 hyphen。
- 删除太泛的词：`software`, `tool`, `app`, `github`, `project`, `ai`。
- 删除重复词：`readme` 和 `github-readme` 可以共存；`open-source` 和 `opensource` 只留一个。
- 不为追热门添加不真实 topic。

每个 topic 给出星级和理由：

| 星级 | 含义 |
|------|------|
| 5 星 | 强相关、高搜索价值、应加入 |
| 4 星 | 相关且有发现价值，可加入 |
| 3 星 | 有一定关系，但不够核心 |
| 2 星 | 相关性弱，通常不推荐 |
| 1 星 | 噪音，不加入 |

输出格式：

```text
GitHub Description 推荐：

1. ★★★★★ <description>
   理由：...

2. ★★★★☆ <description>
   理由：...

最终推荐：<description>

GitHub Topics 推荐：

加入：
- ★★★★★ github-readme — 精准描述项目用途
- ★★★★★ agent-skill — 符合项目形态
- ★★★★☆ image-generation — 当项目支持 Image Gen 时加入

不建议加入：
- ★★☆☆☆ ai — 太泛，搜索噪音大
- ★★☆☆☆ github — 太泛，不能帮助分类
```

### 可选 gh CLI 更新

如果用户安装了 `gh`，当前目录已经是 Git 仓库，并且能解析 GitHub remote，可以建议用户让 Agent 用 `gh repo edit` 更新。不要擅自更新；必须先给出将执行的内容并等待用户确认。

检查：

```bash
gh --version
git remote -v
```

建议命令格式：

```bash
gh repo edit OWNER/REPO \
  --description "<final-description>" \
  --add-topic topic-one \
  --add-topic topic-two \
  --add-topic topic-three
```

如果需要替换旧 topics，先读取当前 topics，再只移除明确不推荐的项：

```bash
gh repo view OWNER/REPO --json description,repositoryTopics
gh repo edit OWNER/REPO --remove-topic old-topic --add-topic new-topic
```

### 说明：git 不能更新仓库元信息

`git` 命令不能修改 GitHub 仓库的 Description 或 Topics。原因：Description 和 Topics 是 GitHub 平台元信息，不是 Git 仓库里的 commit、branch、tag 或 remote 配置。

`git remote -v` 只能用来识别 `OWNER/REPO`，不能用来更新元信息。

如果没有 `gh`，只给出推荐值，让用户到 GitHub 页面手动更新；不要提供复杂备选命令。

不要用 README 文件保存这些推荐；它们属于交付时对用户的操作建议。

---

## Phase 6: 验证和交付

交付前必须验证：

```bash
node --version
npm run showcase
file assets/banner.webp assets/features.webp
git status --short
```

如果改了模板或实际图片，必须重新生成 WebP 并打开检查。

最终汇报只说清楚：

- 改了什么。
- 结果怎样。
- 哪些验证跑过。
- 是否还有需要用户决定的点。

不要把实现细节和冗长过程写给用户。
