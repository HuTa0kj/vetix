package runner

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	einolangsmith "github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/cloudwego/eino/callbacks"
	"github.com/projectdiscovery/gologger"

	"vetix/internal/config"
)

const defaultEndpoint = "https://api.smith.langchain.com"

// newTaskID 同时充当 LangSmith 的 trace id（见 rootRunIDGen），必须是合法 UUID。
func newTaskID() string { return newUUID() }

// rootRunIDGen 让第一个 span 用 Thread ID 作为 id。LangSmith 以根 span 的 id 作为
// 整条 trace 的 id，这样终端打印的 Thread ID 能直接粘进 LangSmith 检索。回调并发
// 调用，但根 span 必定先于任何节点创建，sync.Once 即可。
func rootRunIDGen(taskID string) func(context.Context) string {
	var once sync.Once
	return func(context.Context) string {
		var first string
		once.Do(func() { first = taskID })
		if first != "" {
			return first
		}
		return newUUID()
	}
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// registerTracing 挂上 LangSmith 回调，返回给 run context 打标记的函数；未启用时
// 返回 nil。返回值必须作用到 workflow 的 ctx 上，否则 project 不生效——handler 从
// ctx 读 session name，读不到就留空，run 会被 LangSmith 归到 "default" 项目。
func registerTracing(cfg *config.Config, taskID string) func(context.Context) context.Context {
	ls := cfg.LangSmith
	if ls == nil || ls.APIKey == "" {
		return nil
	}
	if ls.Tracing != nil && !*ls.Tracing {
		return nil
	}

	handler, err := einolangsmith.NewLangsmithHandler(&einolangsmith.Config{
		APIKey:   ls.APIKey,
		APIURL:   ls.Endpoint,
		RunIDGen: rootRunIDGen(taskID),
	})
	if err != nil {
		gologger.Warning().Msgf("LangSmith tracing disabled: %v", err)
		return nil
	}
	callbacks.AppendGlobalHandlers(handler)

	endpoint := ls.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	project := ls.Project
	if project == "" {
		project = "default"
	}
	// endpoint 写错不会让扫描失败，只在每个 span 上静默报错，所以打印生效值。
	gologger.Info().Msgf("LangSmith tracing enabled: endpoint=%s project=%s", endpoint, project)

	return func(ctx context.Context) context.Context {
		return einolangsmith.SetTrace(ctx, einolangsmith.WithSessionName(ls.Project))
	}
}
