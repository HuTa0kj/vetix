package llm

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"vetix/internal/config"
	"vetix/internal/sandbox"
)

// Factory 按角色构造并缓存模型实例，避免同一次扫描里重复建连接。
type Factory struct {
	cfg   *config.Config
	cache map[string]*openai.ChatModel
}

func NewFactory(cfg *config.Config) *Factory {
	return &Factory{cfg: cfg, cache: map[string]*openai.ChatModel{}}
}

// Role 按 config.yaml 的 roles 映射构造 OpenAI 兼容模型。extra_body 原样透传，
// 所以 thinking 这类网关私有参数可以直接写在配置里。
func (f *Factory) Role(ctx context.Context, role string) (*openai.ChatModel, *config.Model, error) {
	m, err := f.cfg.ModelForRole(role)
	if err != nil {
		return nil, nil, err
	}
	m = m.WithThinking(role)
	if cm, ok := f.cache[role]; ok {
		return cm, m, nil
	}

	extra := map[string]any{}
	for k, v := range m.ExtraBody {
		extra[k] = v
	}
	if m.Thinking != nil && *m.Thinking {
		if _, ok := extra["thinking"]; !ok {
			extra["thinking"] = map[string]any{"type": "enabled"}
		}
	}

	cc := &openai.ChatModelConfig{
		APIKey:      m.APIKey,
		BaseURL:     m.BaseURL,
		Model:       m.ID,
		ExtraFields: extra,
	}
	if m.Temperature != nil {
		cc.Temperature = m.Temperature
	}
	cm, err := openai.NewChatModel(ctx, cc)
	if err != nil {
		return nil, nil, fmt.Errorf("create model for role %s: %w", role, err)
	}
	f.cache[role] = cm
	return cm, m, nil
}

// AgentOptions 描述一次 agent 构造所需的全部输入。
type AgentOptions struct {
	Name string
	// Instruction 非空时完全替换 deep 的内建 system prompt。
	Instruction string
	// Tools 是除 eino 内置文件工具外额外注册的工具。
	Tools []tool.BaseTool
	// ReturnDirectly 里的工具一旦被调用，agent 立刻结束本轮并返回该工具结果。
	ReturnDirectly []string
	// HiddenTools 不会出现在模型可见的工具列表里。注意这只是可见性：eino 的
	// 工具派发按 ToolsConfig.Tools 查名字，不校验是否为可见列表的子集，所以
	// 真正的执行边界在 sandbox.Backend 上。
	HiddenTools []string
	// BackendRoot 是虚拟文件系统的根（skill 的父目录）。
	BackendRoot string
	// Allow 是允许读取的虚拟路径前缀。
	Allow []string
	// SkillFS 提供 /skills/** 下的内嵌 helper skill。
	SkillFS fs.FS
	// MaxIters 是模型调用上限，对应 Python 的 ModelCallLimitMiddleware(run_limit)。
	MaxIters int
}

// NewAgent 构造带只读沙箱的 deep agent，等价 Python 的 create_deep_agent。
func (f *Factory) NewAgent(ctx context.Context, role string, o AgentOptions) (adk.ResumableAgent, *config.Model, error) {
	cm, m, err := f.Role(ctx, role)
	if err != nil {
		return nil, nil, err
	}

	cfg := &deep.Config{
		Name:         o.Name,
		ChatModel:    cm,
		Instruction:  o.Instruction,
		Backend:      sandbox.New(o.BackendRoot, o.Allow, o.SkillFS),
		MaxIteration: o.MaxIters,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: o.Tools,
				// 工具错误降级成可自纠的文本结果，避免一次越界读或一个非法正则
				// 就让整轮 agent 作废。
				ToolCallMiddlewares: []compose.ToolMiddleware{toolErrorAsMessage()},
			},
			ReturnDirectly: nameSet(o.ReturnDirectly),
		},
	}
	if len(o.HiddenTools) > 0 {
		cfg.Handlers = append(cfg.Handlers, hideTools(o.HiddenTools))
	}
	// 模型重试只覆盖模型调用；Python 的 ToolRetryMiddleware 重试的是工具调用，
	// 而这里的工具都是本地只读文件操作，重试价值低，不额外实现。
	cfg.ModelRetryConfig = &adk.ModelRetryConfig{MaxRetries: 3}

	agent, err := deep.New(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create agent %s: %w", o.Name, err)
	}
	return agent, m, nil
}

func nameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}
