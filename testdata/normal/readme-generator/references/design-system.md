# README 设计系统

## 色彩主题

默认采用黑底、极简、电影打光、高对比、大留白的杂志封面方向。项目本身没有强品牌色时，不再套多套主题。

```css
--bg-primary:   #050505   /* 极深黑背景 */
--bg-secondary: #0c0b09   /* 低亮度面板 */
--border:       #2a251b   /* 暖暗边界 */
--text-primary: #f4f0e8   /* 暖白主文字 */
--text-muted:   #9a9488   /* 灰色辅助文字 */
--accent:       #c8a15a   /* 暖金强调 */
```

只使用白、灰、暖金三色。不要引入彩虹渐变、紫蓝霓虹或高饱和装饰色。

## 字体层级

```
H1 项目名：92px+ / 800 weight / 0 letter-spacing
中文主标题：80px+ / 800 weight / 0 letter-spacing
H2 章节标题：48px+ / 800 weight / 0 letter-spacing
卡片标题：48px+ / 700 weight
正文：28px+ / 400-600 weight
标签/角标：22px+ / 600 weight / uppercase
```

不要使用 18px 以下文字。GitHub 会缩小图片显示，小字会直接失效。

## 图片文字策略

图片是项目名片，不是无字氛围图，也不是长文信息图。

| 图片 | 必须承担 | 推荐文字 |
|------|----------|----------|
| banner.webp | 项目名和定位 | 项目名 + 一句话定位 + 1-3 个短标签 |
| features.webp | 核心结果和记忆点 | 2-3 个结果短语，必要时加一个短标题 |

文字上限：

- `banner.webp` 不超过 18 个英文词或 28 个中文字。
- `features.webp` 每个短语 2-5 个词。
- 不放段落、命令、表格、密集标签和小字脚注。

文字应该服务传播和定位。读者只看图，也应该知道项目叫什么、解决什么问题、为什么值得继续看。

## 16:9 视觉资产规格

| 图片 | 尺寸 | 用途 |
|------|------|------|
| banner.webp | 16:9 | README 顶部项目名片 |
| features.webp | 16:9 | 核心能力、结果或必要流程 |

默认最多两张图，全部存放在 `assets/` 目录，在 README 中以 `width="100%"` 嵌入。

Codex Image Gen 产物只要求保持 16:9 和足够清晰。README 展示图优先使用 WebP；`1920×1080` 是 HTML to PNG fallback 的模板尺寸，不是 Image Gen 必须尺寸。

不要默认生成单独的流程图。工作方式优先用正文说明；如果必须可视化流程，把它合并进 `features.webp`。

## Badge 规范

```markdown
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)]()
[![Node](https://img.shields.io/badge/Node.js-≥18-6b7c5e.svg)](https://nodejs.org)
```

颜色建议：
- License：`#3b82f6`（蓝）
- Version：`#10b981`（绿）
- Platform/Runtime：`#6b7280`（灰）
- Status：`#f59e0b`（橙，beta）/ `#10b981`（绿，stable）

最多 3 个 badge，保持视觉整洁。

## 作者区块模板

```markdown
## 关于作者

| | |
|:---|:---|
| 个人主页 | [domain.dev](https://domain.dev) |
| GitHub | [username](https://github.com/username) |
```

作者区块只保留最重要入口。纯文字标签，不使用 emoji shortcode（与零 Emoji 原则一致）。
