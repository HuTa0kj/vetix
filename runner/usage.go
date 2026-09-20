package runner

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"

	"vetix/internal/audit"
)

// usageCollector 通过模型组件的回调累计整轮扫描的 token 用量。回调对单次模型调用
// 只触发一次（openai 组件自带回调时 eino 不再在图节点层重复注入），且 OnEnd 携带
// 的 *model.CallbackOutput 自带 TokenUsage，图节点层的转换产物没有这个字段，所以
// 只认非空的 TokenUsage 不会重复计数。
//
// verify 与 behavioral 两条分支并行调用回调，计数走锁。
type usageCollector struct {
	mu    sync.Mutex
	usage audit.TokenUsage
}

func newUsageCollector() *usageCollector {
	return &usageCollector{}
}

// Handler 构造回调 handler。只注册 OnEnd：非流式调用（本项目的全部模型路径，ADK
// 未开 EnableStreaming）成功结束时框架都会带着 *model.CallbackOutput 打到这里；
// 未注册的时序由 TimingChecker 跳过，不给组件调用增加流拷贝开销。
func (c *usageCollector) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			mo := model.ConvCallbackOutput(output)
			if mo == nil || mo.TokenUsage == nil {
				return ctx
			}
			u := mo.TokenUsage
			c.mu.Lock()
			defer c.mu.Unlock()
			c.usage.PromptTokens += u.PromptTokens
			c.usage.CompletionTokens += u.CompletionTokens
			c.usage.ReasoningTokens += u.CompletionTokensDetails.ReasoningTokens
			c.usage.TotalTokens += u.TotalTokens
			c.usage.ModelCalls++
			return ctx
		}).
		Build()
}

// snapshot 取当前累计值。只在 report 节点渲染前调用一次。
func (c *usageCollector) snapshot() audit.TokenUsage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usage
}
