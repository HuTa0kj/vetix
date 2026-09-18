package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// hideTools 从模型可见的工具列表里移除指定工具。
//
// 这只是可见性约束：eino 的工具派发按 ToolsConfig.Tools 查名字执行，不校验
// 调用是否落在可见列表内。真正拦得住模型幻觉调用的是 sandbox.Backend 的
// 只读与白名单检查。
func hideTools(names []string) adk.ChatModelAgentMiddleware {
	hidden := make(map[string]bool, len(names))
	for _, n := range names {
		hidden[n] = true
	}
	return &hideToolsMW{TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[*schema.Message]{}, hidden: hidden}
}

type hideToolsMW struct {
	*adk.TypedBaseChatModelAgentMiddleware[*schema.Message]
	hidden map[string]bool
}

func (h *hideToolsMW) BeforeModelRewriteState(
	ctx context.Context,
	state *adk.ChatModelAgentState,
	mc *adk.ModelContext,
) (context.Context, *adk.ChatModelAgentState, error) {
	kept := state.ToolInfos[:0]
	for _, ti := range state.ToolInfos {
		if h.hidden[ti.Name] {
			continue
		}
		kept = append(kept, ti)
	}
	state.ToolInfos = kept
	return ctx, state, nil
}

// toolErrorAsMessage 把工具执行错误转成工具结果文本回喂给模型。
//
// eino 的默认行为是把工具错误当成节点失败向上抛，整轮 agent 直接终止。对只读
// 文件工具来说终止完全没有必要——grep 传了 RE2 不支持的正则、读了不存在的路径，
// 都是模型自己就能改的。这里统一降级成可自纠的文本结果，并保留原始原因。
func toolErrorAsMessage() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				out, err := next(ctx, input)
				if err == nil {
					return out, nil
				}
				return &compose.ToolOutput{
					Result: fmt.Sprintf("tool %q failed: %v", input.Name, err),
				}, nil
			}
		},
	}
}
