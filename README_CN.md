# Vetix

基于 LLM Agent 的 [SKILL](https://docs.claude.com/en/docs/claude-code/skills) 目录安全扫描工具。Vetix 把确定性的插件规则与 LLM 行为分析结合在一起，在同一次扫描中既能抓到明显的攻击特征，也能发现隐蔽、混淆的攻击链。

[English](./README.md)

## 功能特性

- **基于插件的静态扫描** —— 通过规则识别确定性安全风险。
- **LLM 交叉校验** —— 每个插件命中的风险项都会被 LLM 结合真实文件内容再次判断，避免高召回规则淹没最终报告。
- **行为分析 Agent** —— 在虚拟文件系统下完整追踪「指令 → 工具调用 → 对主机的影响」执行链，发现规则无法识别的风险：伪装命令、Base64 载荷、远程代码加载、提示词注入、凭据窃取、持久化等。
- **纵深防御沙箱** —— 虚拟文件系统、显式读允许列表、全局写拒绝，隔离真实主机。
- **LangSmith 追踪** —— 端到端可观测每一次 Agent 运行。

## 检测分类

行为分析 Agent 将风险分为 11 个类别：

| 分类 | 描述 |
|---|---|
| 远程执行 | 远程代码加载与执行，包括 `curl\|sh`、`wget\|bash` 以及非官方源的包安装 |
| 数据外泄 | 未经授权收集敏感数据并传输至外部地址 |
| 凭据窃取 | 窃取或暴露 API 密钥、Token、密码等凭据 |
| 持久化 | 通过 crontab 注入、SSH 密钥植入、启动项篡改等手段实现持久化 |
| 破坏性操作 | 破坏数据、删除文件或对主机系统造成其他损害的行为 |
| 代码混淆 | 通过 Base64/Hex 编码、大量空行隐藏、伪装命令等手段刻意隐藏恶意载荷 |
| 命令注入 | 通过未清洗的输入或指令操控注入任意 Shell 命令 |
| 权限提升 | 试图获取超出 SKILL 声明功能所需的更高权限 |
| 敏感文件访问 | 未经授权读取或写入 SSH 密钥、`.aws` 凭据、浏览器数据、`.env` 文件等敏感信息 |
| 网络滥用 | 可疑的外发连接、C2 通信或指向硬编码外部 IP/域名的流量 |
| 提示词注入 | 改写 Agent 行为的指令——「忽略之前的指令」「DAN 模式」「忘记一切」等 |

## 为什么选择 Agent？

传统的基于规则的扫描器依赖预定义的模式和签名，无法有效检测新型或隐蔽威胁。Vetix 利用 LLM 驱动的智能体突破这些限制：

- **超越规则** —— Agent 能理解代码语义与意图，发现基于规则的方法漏掉的恶意行为（混淆代码、多步攻击链、上下文相关漏洞利用）。
- **自适应推理** —— 与静态规则不同，Agent 能对未知代码模式进行动态推理，并根据扫描过程中的发现自适应调整策略。
- **上下文感知分析** —— Agent 在整个 SKILL 的全局上下文中评估风险，识别单条规则无法捕获的跨文件交互和链式漏洞。
- **自然语言解释** —— 每一项发现都附带清晰、易读的风险说明、影响评估和修复建议，而不仅仅是一个规则编号。

## 快速开始

### 环境要求

- Python ≥ 3.12
- [uv](https://docs.astral.sh/uv/)（推荐包管理工具）

### 安装

```bash
git clone git@github.com:HuTa0kj/vetix.git
cd vetix
uv sync
```

### 配置

复制示例配置文件并填写模型凭据：

```bash
cp example.config.yaml config.yaml
```

`config.yaml` 需要定义两个 LLM 角色：一个轻量模型用于插件命中校验，一个强模型用于行为分析。

```yaml
models:
  - id: deepseek-v4-pro
    name: DeepSeek-V4-Pro
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {"thinking": {"type": "disabled"}}

  - id: deepseek-v4-flash
    name: DeepSeek-V4-Flash
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {"thinking": {"type": "disabled"}}

roles:
  lite: deepseek-v4-flash
  pro:  deepseek-v4-pro

# 可选：LangSmith 追踪
langsmith:
  tracing: true
  endpoint: "https://api.smith.langchain.com"
  api_key: ""
  project: ""
```

| 字段 | 说明 |
|------|------|
| `models` | 可用 LLM 列表。每项需配置 `id`、`api_key`、`base_url`；`temperature`、`extra_body` 可选。 |
| `roles.lite` | 快速模型，适用于追求速度、不复杂的任务。 |
| `roles.pro` | 推理模型，适用于需要复杂推理的任务。 |
| `langsmith` | LangSmith 追踪配置（可选）。 |

### 使用

```bash
# 扫描指定 SKILL 目录
uv run vetix scan --source ~/.claude/skills/skill-directory

# 简写
uv run vetix scan -s ./examples/skills/malicious/pop-calc

# 开启调试日志
uv run vetix scan -s ./examples/skills/malicious/pop-calc --debug

# 使用中文输出
uv run vetix scan -s ./examples/skills/malicious/pop-calc -l zh

# 创建新插件
uv run vetix create --plugin "my check"
```

## Agent 追踪

在 `config.yaml` 中配置 [LangSmith](https://smith.langchain.com/) 后，可以追踪每一次 Agent 运行——模型调用、工具调用和结构化输出全程可见。

![](./images/langsmith.png)

## License

[MIT](LICENSE)
