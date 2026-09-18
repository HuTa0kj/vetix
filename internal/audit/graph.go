package audit

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"vetix/internal/llm"
)

// 节点名同时出现在流程图与调试日志里。
const (
	nodeGather   = "gather_base_info"
	nodePlugins  = "plugins_check"
	nodeVerify   = "plugins_findings_verify"
	nodeBehavior = "behavioral_analysis"
	nodeReport   = "report"
)

// Workflow 构建并编译审计工作流。
//
// 拓扑：gather → plugins → {verify, behavioral} → report。
//
// 图的数据类型统一用 map[string]any：compose 的 fan-in 合并函数只内置支持 map
// （以及显式注册过的类型），report 有两个前驱，用标量类型会在汇聚点直接报
// "unsupported type"。节点间本来就不靠边传值，全部走共享 state。
func Workflow(ctx context.Context, f *llm.Factory, report func(*State) error) (compose.Runnable[map[string]any, map[string]any], error) {
	g := compose.NewGraph[map[string]any, map[string]any](
		compose.WithGenLocalState(func(ctx context.Context) *State { return stateFrom(ctx) }))

	add := func(key string, fn func(context.Context, *stateHandle) error) error {
		return g.AddLambdaNode(key, compose.InvokableLambda(
			func(ctx context.Context, _ map[string]any) (map[string]any, error) {
				if err := fn(ctx, &stateHandle{ctx: ctx}); err != nil {
					return nil, err
				}
				return map[string]any{key: true}, nil
			}),
			// 无它 RunInfo.Name 为空，节点在追踪里全叫 "Lambda"。
			compose.WithNodeName(key))
	}

	if err := add(nodeGather, func(_ context.Context, h *stateHandle) error { return GatherBaseInfo(h) }); err != nil {
		return nil, err
	}
	if err := add(nodePlugins, func(_ context.Context, h *stateHandle) error { return PluginCheck(h) }); err != nil {
		return nil, err
	}
	if err := add(nodeVerify, func(ctx context.Context, h *stateHandle) error { return VerifyFindings(ctx, h, f) }); err != nil {
		return nil, err
	}
	if err := add(nodeBehavior, func(ctx context.Context, h *stateHandle) error { return BehavioralAnalysis(ctx, h, f) }); err != nil {
		return nil, err
	}
	if err := add(nodeReport, func(_ context.Context, h *stateHandle) error {
		var runErr error
		if err := h.with(func(s *State) { runErr = report(s) }); err != nil {
			return err
		}
		return runErr
	}); err != nil {
		return nil, err
	}

	for _, e := range [][2]string{
		{compose.START, nodeGather},
		{nodeGather, nodePlugins},
		{nodePlugins, nodeVerify},
		{nodePlugins, nodeBehavior},
		{nodeVerify, nodeReport},
		{nodeBehavior, nodeReport},
		{nodeReport, compose.END},
	} {
		if err := g.AddEdge(e[0], e[1]); err != nil {
			return nil, err
		}
	}

	// 必须显式指定 AllPredecessor。默认的 AnyPredecessor 是 pregel 语义：任一
	// 前驱产出就会触发 report，两条并行分支会有一条的结果还没写完。
	return g.Compile(ctx,
		compose.WithGraphName("skill_safe_audit"),
		compose.WithNodeTriggerMode(compose.AllPredecessor))
}

// stateHandle 是节点函数访问共享 state 的入口，每次 with 取一次锁、绝不长持。
// compose.ProcessState 持锁到 handler 返回，而 verify 与 behavioral 是并行的，
// 在锁内跑 agent 会让先抢到锁的一条独占几十秒，另一条从节点启动起就干等。
type stateHandle struct{ ctx context.Context }

func (h *stateHandle) with(fn func(*State)) error {
	return compose.ProcessState(h.ctx, func(_ context.Context, s *State) error {
		fn(s)
		return nil
	})
}

func (h *stateHandle) snapshot() (snapshot, error) {
	var out snapshot
	err := h.with(func(s *State) { out = s.snapshot() })
	return out, err
}

type seedKey struct{}

// WithSeed 把本次扫描的初始 State 挂到 context 上，供 state 生成器取出。
func WithSeed(ctx context.Context, s *State) context.Context {
	return context.WithValue(ctx, seedKey{}, s)
}

func stateFrom(ctx context.Context) *State {
	if s, ok := ctx.Value(seedKey{}).(*State); ok {
		return s
	}
	return &State{}
}
