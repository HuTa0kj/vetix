package runner

import (
	"crypto/rand"
	"encoding/hex"

	einolangsmith "github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/cloudwego/eino/callbacks"
	"github.com/projectdiscovery/gologger"

	"vetix/internal/config"
)

func newTaskID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

// registerTracing 按配置挂上 LangSmith 回调。eino 的 span 结构与 LangChain 不同，
// 追踪记录在 LangSmith 界面上的呈现形态也与之有别。
func registerTracing(cfg *config.Config) {
	ls := cfg.LangSmith
	if ls == nil || ls.APIKey == "" {
		return
	}
	if ls.Tracing != nil && !*ls.Tracing {
		return
	}
	handler, err := einolangsmith.NewLangsmithHandler(&einolangsmith.Config{
		APIKey: ls.APIKey,
		APIURL: ls.Endpoint,
	})
	if err != nil {
		gologger.Warning().Msgf("LangSmith tracing disabled: %v", err)
		return
	}
	callbacks.AppendGlobalHandlers(handler)
}
