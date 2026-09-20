package runner

import (
	"context"
	"sync"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"

	"vetix/internal/audit"
)

func feedUsage(c *usageCollector, n int, prompt, completion, reasoning, total int) {
	h := c.Handler()
	for i := 0; i < n; i++ {
		h.OnEnd(context.Background(), nil, &model.CallbackOutput{
			TokenUsage: &model.TokenUsage{
				PromptTokens:            prompt,
				CompletionTokens:        completion,
				TotalTokens:             total,
				CompletionTokensDetails: model.CompletionTokensDetails{ReasoningTokens: reasoning},
			},
		})
	}
}

func TestUsageCollectorAccumulates(t *testing.T) {
	c := newUsageCollector()
	feedUsage(c, 2, 100, 50, 20, 150)
	feedUsage(c, 1, 10, 5, 0, 15)

	got := c.snapshot()
	want := audit.TokenUsage{
		PromptTokens: 210, CompletionTokens: 105,
		ReasoningTokens: 40, TotalTokens: 315, ModelCalls: 3,
	}
	if got != want {
		t.Fatalf("usage = %+v, want %+v", got, want)
	}
}

// 图节点层的回调转换产物（*schema.Message 经 ConvCallbackOutput 得到的
// CallbackOutput）没有 TokenUsage，组件自身发的才有。只认非空 TokenUsage 才不会
// 把同一个模型调用算两遍。
func TestUsageCollectorIgnoresOutputWithoutUsage(t *testing.T) {
	c := newUsageCollector()
	h := c.Handler()
	for _, out := range []callbacks.CallbackOutput{
		&model.CallbackOutput{Message: nil, TokenUsage: nil},
		"not a model output at all",
		nil,
	} {
		h.OnEnd(context.Background(), nil, out)
	}
	if got := c.snapshot(); got != (audit.TokenUsage{}) {
		t.Fatalf("usage = %+v, want zero value", got)
	}
}

func TestUsageCollectorConcurrent(t *testing.T) {
	c := newUsageCollector()
	h := c.Handler()

	const goroutines, perG = 8, 100
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				h.OnEnd(context.Background(), nil, &model.CallbackOutput{
					TokenUsage: &model.TokenUsage{PromptTokens: 1, TotalTokens: 1},
				})
			}
		}()
	}
	wg.Wait()

	want := audit.TokenUsage{PromptTokens: goroutines * perG, TotalTokens: goroutines * perG, ModelCalls: goroutines * perG}
	if got := c.snapshot(); got != want {
		t.Fatalf("usage = %+v, want %+v", got, want)
	}
}
