# Vetix

基于 LLM Agent 的 [SKILL](https://docs.claude.com/en/docs/claude-code/skills) 目录安全扫描工具。Vetix 把确定性的插件规则与 LLM 行为分析结合在一起，在同一次扫描中既能抓到明显的攻击特征，也能发现隐蔽、混淆的攻击链。

[English](./README.md)

> [!WARNING]
> [`examples/`](./examples) 目录下为用于测试的示例 SKILL，其中包含刻意构造的恶意样本。请勿在扫描测试环境之外安装或加载这些 SKILL。

## 功能特性

- **基于插件的静态扫描** —— 通过规则识别确定性安全风险。
- **LLM 交叉校验** —— 每个插件命中的风险项都会被 LLM 结合真实文件内容再次判断，避免高召回规则淹没最终报告。
- **行为分析 Agent** —— 在只读虚拟文件系统下完整追踪「指令 → 工具调用 → 对主机的影响」执行链，发现规则无法识别的风险：伪装命令、Base64 载荷、远程代码加载、提示词注入、敏感文件访问、持久化等。
- **纵深防御沙箱** —— Agent 只能读到 SKILL 自身目录，符号链接一律拒绝，写入在后端层被整体拒绝，写类工具也不出现在模型可见的工具列表里。
- **Token 用量统计** —— 报告记录整轮扫描全部模型调用的 token 用量（prompt / completion / 总量 / 调用次数），便于估算批量扫描成本。
- **LangSmith 追踪** —— 端到端可观测每一次 Agent 运行。

## 参数

```
Usage:
  vetix -s <skill-dir> | -p <preset> [flags]

Flags:
  INPUT
    -s, -source string   SKILL directory path or a skills parent directory
    -p, -preset string   Scan all skills under a preset agent skills root (claude-code, codex)

  CONFIG
    -c, -config string   Path to the YAML config file (default "./config.yaml")
    -l, -language string Output language for audit findings (en, zh) (default "en")

  OUTPUT
    -o, -output          Save the audit report to <output-dir>/<skill-hash-prefix>/report.json (default true)
    -no-output           Disable saving the audit report to a JSON file
    -output-dir string   Base directory for saved reports (default "./output")
    -force               Ignore the cached report and force a full re-scan

  DEBUG
    -d, -debug           Enable debug logging

  PLUGIN
    -pl, -plugins-list   List built-in plugins with their names and descriptions
```

## 常用命令

```bash
# 检测单个 SKILL，目录下需要有 SKILL.md
vetix -s ./examples/malicious/wacli-1sk

# 扫描整个 SKILL 目录，直接子目录下需要有 SKILL.md
vetix -s ~/.claude/skills

# 使用预设路径扫描
vetix -p claude-code

# 查看插件列表
vetix -pl
```

## 检测分类

行为分析 Agent 将风险分为 10 个类别：

| 分类 | 描述 |
|---|---|
| 远程执行 | 远程代码加载与执行，包括 `curl\|sh`、`wget\|bash` 以及非官方源的包安装 |
| 数据外泄 | 未经授权收集敏感数据并传输至外部地址 |
| 持久化 | 通过 crontab 注入、SSH 密钥植入、启动项篡改等手段实现持久化 |
| 破坏性操作 | 破坏数据、删除文件或对主机系统造成其他损害的行为 |
| 代码混淆 | 通过 Base64/Hex 编码、大量空行隐藏、伪装命令等手段刻意隐藏恶意载荷 |
| 命令注入 | 通过未清洗的输入或指令操控注入任意 Shell 命令 |
| 权限提升 | 试图获取超出 SKILL 声明功能所需的更高权限 |
| 敏感文件访问 | 未经授权读取或写入 SSH 密钥、`.aws` 凭据、API 密钥、Token、密码、浏览器数据、`.env` 文件等敏感信息 |
| 网络滥用 | 可疑的外发连接、C2 通信或指向硬编码外部 IP/域名的流量 |
| 提示词注入 | 改写 Agent 行为的指令——「忽略之前的指令」「DAN 模式」「忘记一切」等 |

## 为什么选择 Agent？

传统的基于规则的扫描器依赖预定义的模式和签名，无法有效检测新型或隐蔽威胁。Vetix 利用 LLM 驱动的智能体突破这些限制：

- **超越规则** —— Agent 能理解代码语义与意图，发现基于规则的方法漏掉的恶意行为（混淆代码、多步攻击链、上下文相关漏洞利用）。
- **自适应推理** —— 与静态规则不同，Agent 能对未知代码模式进行动态推理，并根据扫描过程中的发现自适应调整策略。
- **上下文感知分析** —— Agent 在整个 SKILL 的全局上下文中评估风险，识别单条规则无法捕获的跨文件交互和链式漏洞。
- **自然语言解释** —— 每一项发现都附带清晰、易读的风险说明、影响评估和修复建议，而不仅仅是一个规则编号。

## 部署运行

### 编译

需要 Go 1.25 及以上：

```bash
git clone git@github.com:HuTa0kj/vetix.git
cd vetix
go build -o vetix ./cmd/vetix
```

复制配置模板并填入模型凭据：

```bash
cp example.config.yaml config.yaml
```

`config.yaml` 默认从当前工作目录读取，可用 `-config` 指定其他路径。它定义了两个 LLM 角色：用于插件命中复核的轻量模型，以及用于行为分析的更强模型。

```yaml
models:
  - id: deepseek-v4-pro
    name: DeepSeek-V4-Pro
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {}
    extra_headers: {}       # 附加到每一次请求（快速路径与 agent 内部调用都生效）
    thinking: true          # 在请求体里注入 {"thinking": {"type": "enabled"}}

  - id: deepseek-v4-flash
    name: DeepSeek-V4-Flash
    api_key: ""
    base_url: "https://example.com/v1"
    temperature: 0.7
    extra_body: {}
    thinking: false

roles:
  lite: deepseek-v4-flash
  pro:  deepseek-v4-pro

# 可选：LangSmith 追踪
langsmith:
  tracing: false
  endpoint: "https://api.smith.langchain.com"
  api_key: ""
  project: ""
```

| 字段 | 说明 |
|-------|-------------|
| `models` | 可用模型列表。每条必须提供 `id`、`api_key`、`base_url`；`temperature`、`extra_body`、`extra_headers`、`thinking`、`response_format` 可选。`base_url` 要指向 API 根路径，多数网关需要带 `/v1`，否则会返回网页而不是 JSON。 |
| `models[].extra_headers` | 随每次请求发送的额外 HTTP 头，例如网关的路由标识。在传输层注入，agent 内部的模型调用同样生效。 |
| `models[].thinking` | 是否请求思考。默认 `pro` 角色开启、`lite` 角色关闭。注意部分网关在思考模式下会拒绝强制工具调用，单文件快速路径会识别这种拒绝并自动改用 `response_format`。 |
| `models[].response_format` | `tool`（默认）用强制具名工具调用来拿结构化输出；`json_schema` 使用网关原生的 `response_format`。 |
| `roles.lite` | 轻量模型，用于插件命中复核。 |
| `roles.pro` | 推理模型，用于行为分析。 |
| `langsmith` | LangSmith 追踪配置（可选） |

### Docker

构建镜像

```bash
docker build -t vetix:latest .
```

准备配置

```bash
cp example.config.yaml config.yaml
# 编辑 config.yaml，填入两个模型的真实 api_key / base_url
```

执行扫描

```bash
docker run --rm \
  -v "$PWD/config.yaml:/work/config.yaml:ro" \
  -v "$PWD/examples/skills/xxx:/skills/xxx:ro" \
  -v "$PWD/output:/work/output" \
  vetix:latest -s /skills/xxx
```

### Docker Compose

```bash
docker compose run --rm vetix -s /skills/xxx
```

## 内置插件

每个插件对 SKILL 目录里的每个文件运行；标记为「LLM 复核」的命中会先由复核阶段结合真实文件内容再次判断，其余直接进入报告。

| 插件 | 检测内容 | 默认严重度 | LLM 复核 |
|---|---|---|---|
| `base64_exec` | Base64 解码后经管道送入 Shell 的命令 | critical | 是 |
| `reverse_shell` | 反弹 Shell 特征——`/dev/tcp`、`nc -e`、`socat exec:` | critical | 是 |
| `binary_file` | SKILL 目录中的二进制文件 | high | 否 |
| `consecutive_newlines` | 30 个以上连续换行隐藏内容 | high | 否 |
| `exceptional_file` | 本应是文本的文件里混入大量不可打印字符 | medium | 否 |
| `large_file` | 单文件超过 2 MB | medium | 否 |
| `long_file` | 单文件超过 3000 行 | medium | 否 |
| `public_ip` | 硬编码的公网 IPv4 地址 | medium | 是 |
| `rare_file` | 扩展名不在文本白名单内的罕见文件 | medium | 否 |

## 新增插件

插件位于 `internal/plugin/`。新增一个实现 `Plugin` 的文件，需要在 `internal/plugin/registry.go` 里注册：

```go
package plugin

type MyCheckPlugin struct{}

func (MyCheckPlugin) Meta() Meta {
    return Meta{
        ID:          "my_check",
        Name:        "My Check",
        Description: "一句话说明这条规则检测什么。",
    }
}

func (MyCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
    // 返回全部命中；把 AuditRequired 设为 true 的命中会走 LLM 复核。
    return nil
}
```

## 许可证

[MIT](LICENSE)
