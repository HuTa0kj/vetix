<div align="center">

# GitHub README Generator

**把项目 README 做成一份可以展示的作品。**

<img src="assets/banner.webp" alt="GitHub README Generator — portfolio-grade README design for open source projects" width="100%">

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Node](https://img.shields.io/badge/Node.js-≥18-6b7c5e.svg)](https://nodejs.org)
[![Skill](https://img.shields.io/badge/Codex_Skill-README_Design-c96442.svg)](#快速开始)

</div>

---

## 这是什么

GitHub README Generator 是一个 Agent Skill，用来把普通项目 README 升级成作品集级项目首页。

它不会只替你填模板，而是先阅读项目，提炼背景、价值和使用路径，再生成克制的文案、作品级视觉资产和 GitHub 元信息建议。

---

## 为什么需要它

GitHub 项目就是开发者的作品集。README 是读者、招聘者、用户和 AI 搜索首先看到的页面。

很多项目明明有价值，却被冗长说明、小字截图、过度装饰和模板化功能列表削弱了第一印象。这个 Skill 的目标是反过来做：少说废话，放大重点，让项目自己的故事站出来。

---

## 你会得到什么

<img src="assets/features.webp" alt="Three outcomes: story, visual identity, and clean delivery" width="100%">

---

## 工作方式

它先阅读项目，判断 README 应该是偏作品展示还是偏清晰文档；再提炼项目故事，生成正文、两张以内的视觉资产和 GitHub 元信息建议。

---

## 快速开始

安装后，在 Codex 中打开你的项目目录，然后直接说：

```text
帮我把这个项目的 README 做成作品集级项目首页
```

也可以指定视觉方式：

```text
用 Codex Image Gen 生成 README 封面，其余保持清晰信息图
```

```text
生成一个极简文档型 README，不要重视觉
```

---

## 安装

```bash
npx skills add https://github.com/geekjourneyx/readme-generator
```

如果需要使用 HTML 模板截图生成图片，请确认本地有 Node.js 和 Playwright：

```bash
node -v
npm install
npx playwright install chromium
```

---

## 视觉模式

| 模式 | 适合项目 | 图片策略 |
|------|----------|----------|
| `portfolio` | 默认推荐，多数开源项目 | 封面可用 Codex Image Gen，说明图用大字海报 |
| `clean-doc` | SDK、库、后端工具 | 少图，优先快速开始和示例 |
| `visual-story` | AI 工具、设计工具、独立产品 | 更强视觉叙事和封面表现 |
| `structured` | 需要流程、能力、对比图 | HTML/CSS 模板截图，保证文字准确 |

---

## 生成内容

```text
project/
├── README.md
├── LICENSE
└── assets/
    ├── banner.webp
    └── features.webp
```

图片可以来自两条路线：

- Codex Image Gen：适合封面、氛围图、作品集视觉。
- HTML 截图模板：适合功能、流程、对比等需要文字准确的图。

---

## 设计标准

这个 Skill 按 100 分 README 标准工作：

| 维度 | 分值 |
|------|------|
| 15 秒理解项目价值 | 25 |
| 讲清项目故事 | 20 |
| 视觉像作品而不是说明书 | 20 |
| 快速开始足够直接 | 15 |
| 有可信产物展示 | 10 |
| 克制、少噪音、不过度设计 | 10 |

---

## 许可证

[MIT](./LICENSE)

---

## 关于作者

| | |
|:---|:---|
| 个人主页 | [geekjourney.dev](https://geekjourney.dev) |
| GitHub | [geekjourneyx](https://github.com/geekjourneyx) |
| Twitter | [@seekjourney](https://x.com/seekjourney) |
| 公众号 | 微信搜「极客杰尼」 |
