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
	// 共享 state 一律通过 compose.ProcessState 访问：compose 在调用回调期间持有
	// per-state 互斥锁，两条并行分支因此是串行安全的。抓指针直接改会绕过这把锁。
	g := compose.NewGraph[map[string]any, map[string]any](
		compose.WithGenLocalState(func(ctx context.Context) *State { return stateFrom(ctx) }))

	add := func(key string, fn func(context.Context, *State) error) error {
		return g.AddLambdaNode(key, compose.InvokableLambda(
			func(ctx context.Context, _ map[string]any) (map[string]any, error) {
				var runErr error
				if err := compose.ProcessState(ctx, func(ctx context.Context, s *State) error {
					runErr = fn(ctx, s)
					return nil
				}); err != nil {
					return nil, err
				}
				if runErr != nil {
					return nil, runErr
				}
				return map[string]any{key: true}, nil
			}))
	}

	if err := add(nodeGather, func(_ context.Context, s *State) error { return GatherBaseInfo(s) }); err != nil {
		return nil, err
	}
	if err := add(nodePlugins, func(_ context.Context, s *State) error { return PluginCheck(s) }); err != nil {
		return nil, err
	}
	if err := add(nodeVerify, func(ctx context.Context, s *State) error { return VerifyFindings(ctx, s, f) }); err != nil {
		return nil, err
	}
	if err := add(nodeBehavior, func(ctx context.Context, s *State) error { return BehavioralAnalysis(ctx, s, f) }); err != nil {
		return nil, err
	}
	if err := add(nodeReport, func(_ context.Context, s *State) error { return report(s) }); err != nil {
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

// StateFrom 取出本次运行共享的 State，供调用方在 run 结束后读取结果。
func StateFrom(ctx context.Context) *State { return stateFrom(ctx) }
