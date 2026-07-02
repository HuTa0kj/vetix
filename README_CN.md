# Vetix

基于 LLM Agent 的 SKILL 安全扫描工具，自动化识别与评估 SKILL 目录中的安全风险。

## 功能特性

- 自动解析 SKILL 目录结构，提取基础信息
- 基于 LLM 生成 SKILL 概述报告
- 自动检测脚本文件并进行代码安全审计
- 支持中英文报告输出
- 集成 LangSmith 追踪
- 终端报告展示 + 文件持久化输出

## 为什么选择 Agent？

传统的基于规则的扫描器依赖预定义的模式和签名，无法有效检测新型或隐蔽的威胁。Skill Scanner Agent 利用 LLM 驱动的智能体来突破这些限制：

- **超越规则** — Agent 能够理解代码语义和意图，检测到基于规则的方法无法发现的恶意行为（如混淆代码、多步骤攻击链、上下文感知的漏洞利用）。
- **自适应推理** — 与静态规则不同，Agent 能够对未知的代码模式进行动态推理，并根据扫描过程中的发现自适应调整分析策略。
- **上下文感知分析** — Agent 能够在 SKILL 的全局上下文中评估安全风险，识别出单条规则无法捕获的跨文件交互和链式漏洞。
- **自然语言解释** — 每一项发现都附带清晰、易读的风险说明、影响评估和修复建议，而不仅仅是一个规则编号。

## 工作流程

1. **gather_base_info** — 验证 SKILL 目录，提取名称，检测是否存在脚本文件
2. **skill_summary** — 通过 LLM Agent 对 SKILL 进行安全概述分析
3. **audit_scripts** — 通过 LLM Agent 对脚本文件进行代码安全审计

## 快速开始

### 环境要求

- Python >= 3.12
- [uv](https://docs.astral.sh/uv/)（推荐包管理工具）

### 安装

```bash
# 克隆仓库
git clone git@github.com:HuTa0kj/vetix.git
cd vetix

# 安装依赖
uv sync
```

### 配置

复制示例配置文件并填写必要信息：

```bash
cp config.yaml.example config.yaml
```

编辑 `config.yaml`，配置模型 API 信息：

```yaml
models:
  - id: glm-5
    name: GLM-5
    api_key: ""
    base_url: ""
    temperature: 0.1

  - id: deepseek-v4-flash
    name: DeepSeek-V4-Flash
    api_key: ""
    base_url: ""
    temperature: 0.1
    extra_body: {"thinking": {"type": "disabled"}}

roles:
  skill_summary: deepseek-v4-flash
  audit_scripts: glm-5

limit:
  model_call: 80
  tool_call: 80

# langsmith config (Optional)
langsmith:
  tracing: true
  endpoint: "https://api.smith.langchain.com"
  api_key: ""
  project: ""

# Script files to be detected
script_extensions: ['.py', '.sh', '.bash', '.js', '.ts', '.rb', '.pl', '.go', '.rs', '.ps1', '.cmd', '.bat']
debug: false
output_dir: "./output"
language: "en"

```

**配置说明：**

| 字段 | 说明 |
|------|------|
| `models` | 可用的 LLM 模型列表，每个模型需配置 `id`、`api_key`、`base_url` |
| `roles` | 角色与模型的映射关系，支持为不同任务指定不同模型 |
| `langsmith` | LangSmith 追踪配置（可选） |
| `script_extensions` | 需要检测的脚本文件扩展名 |
| `output_dir` | 报告输出目录 |
| `language` | 报告语言，支持 `en`（英文）和 `zh`（中文） |

### 使用

```bash
# 扫描指定 SKILL 目录
uv run vetix scan --source ~/.claude/skills/skill-directory
```

扫描目标目录必须包含 `SKILL.md` 文件。

### 输出

扫描完成后，报告将保存在 `output/<task_id>/` 目录下：

```
output/
└── <task_id>/
    ├── skill_summary.md    # SKILL 概述报告
    └── code_audit.md       # 代码安全审计报告（仅当存在脚本文件时）
```

## Agent 追踪

在 `config.yaml` 中配置 [LangSmith](https://smith.langchain.com/) 密钥后，可以追踪 Agent 的执行过程，查看所有工具调用和详细信息。

![](./images/langsmith.png)

## 技术栈

- **LangGraph** — 工作流编排
- **LangChain** — LLM 调用与消息管理
- **DeepAgents** — Agent 构建
- **Typer** — CLI 框架
- **Rich** — 终端美化输出

## License

[MIT](LICENSE)
