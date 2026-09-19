package runner

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"vetix/internal/config"
)

// LangSmith 只接受合法 UUID 作为 trace id，裸十六进制会被拒——而拒绝只表现为
// handler 内部一行日志，扫描照常成功。
var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewTaskIDIsUUID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newTaskID()
		if !uuidRe.MatchString(id) {
			t.Fatalf("not a v4 UUID: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id: %q", id)
		}
		seen[id] = true
	}
}

// Thread ID 必须是根 span 的 id，否则 trace id 与终端打印的 Thread ID 对不上。
func TestRootRunIDGenUsesTaskIDOnce(t *testing.T) {
	const taskID = "11111111-2222-4333-8444-555555555555"
	gen := rootRunIDGen(taskID)

	if got := gen(context.Background()); got != taskID {
		t.Fatalf("first id must be the task id: %q", got)
	}
	for i := 0; i < 10; i++ {
		got := gen(context.Background())
		if got == taskID {
			t.Fatal("task id handed out more than once")
		}
		if !uuidRe.MatchString(got) {
			t.Fatalf("not a v4 UUID: %q", got)
		}
	}
}

func TestRegisterTracingDisabled(t *testing.T) {
	off := false
	for name, cfg := range map[string]*config.Config{
		"no langsmith block": {},
		"no api key":         {LangSmith: &config.LangSmith{}},
		"tracing false": {LangSmith: &config.LangSmith{
			APIKey: "k", Tracing: &off,
		}},
	} {
		handler, traceCtx := registerTracing(cfg, newTaskID())
		if handler != nil || traceCtx != nil {
			t.Errorf("%s: tracing must stay off", name)
		}
	}
}

func TestRegisterTracingProject(t *testing.T) {
	on := true
	cfg := &config.Config{LangSmith: &config.LangSmith{
		APIKey:  "lsv2_pt_test",
		Tracing: &on,
		Project: "vetix",
	}}
	handler, traceCtx := registerTracing(cfg, newTaskID())
	if handler == nil || traceCtx == nil {
		t.Fatal("tracing must be enabled")
	}
	if traceCtx(context.Background()) == nil {
		t.Fatal("decorated context must not be nil")
	}
}

func TestUUIDShapeDiffersFromBareHex(t *testing.T) {
	id := newTaskID()
	if strings.Count(id, "-") != 4 {
		t.Fatalf("expected 4 dashes in %q", id)
	}
}
