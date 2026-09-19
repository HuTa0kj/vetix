package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/projectdiscovery/gologger"

	"vetix/internal/audit"
	"vetix/internal/config"
	"vetix/internal/llm"
	"vetix/internal/plugin"
	"vetix/internal/report"
)

type Runner struct {
	Options *Options
}

func New(options *Options) (*Runner, error) {
	return &Runner{Options: options}, nil
}

func (r *Runner) Run() error {
	plugin.SetWarnSink(func(format string, args ...any) {
		gologger.Warning().Msgf(format, args...)
	})

	ctx := context.Background()
	start := time.Now()

	cfg, err := config.Load(r.Options.ConfigPath)
	if err != nil {
		return err
	}

	source, err := filepath.Abs(r.Options.Source)
	if err != nil {
		return err
	}
	outputDir, err := filepath.Abs(r.Options.OutputDir)
	if err != nil {
		return err
	}

	state := &audit.State{
		TaskID:     newTaskID(),
		SkillDir:   source,
		Workspace:  filepath.Dir(source),
		OutputDir:  outputDir,
		SaveOutput: r.Options.SaveReport(),
		DetectedAt: time.Now().Format("2006-01-02T15:04:05-07:00"),
		Language:   r.Options.Language,
	}
	// Thread ID 同时是 LangSmith 的根 span id。
	gologger.Info().Msgf("Thread ID: %s", state.TaskID)

	if state.SaveOutput {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}
		gologger.Info().Msgf("Output base directory: %s", outputDir)
	}

	// 缓存查找不受 -no-output 影响：只要上次连同报告一起落过盘，这次就直接渲染。
	if !r.Options.Force {
		cached, err := audit.LoadCachedReport(source, outputDir)
		if err != nil {
			return err
		}
		if cached != nil {
			cached.SaveOutput = state.SaveOutput
			report.Render(cached)
			gologger.Info().Msg("Loaded cached report (use -force to re-scan)")
			return nil
		}
	}

	// 缓存命中时 pipeline 不跑，没有 span 可发。
	traceCtx := registerTracing(cfg, state.TaskID)

	factory := llm.NewFactory(cfg)
	// installRuninfoFilter 在 workflow 构建之后才装，这里先声明，render 运行时已就绪。
	var restoreRuninfo func()
	render := func(s *audit.State) error {
		// 渲染前先恢复真实 stdout：日志走 stderr 同步直达终端，而报告要过 runinfo
		// 过滤器的管道、由后台协程异步转发。不恢复的话，报告尾段还压在管道里时
		// "Report saved" 日志就先到了，视觉上插进报告中间。report 是最后一个节点，
		// 此刻全部 LLM span 已结束，此后不会再有 runinfo 记录，提前恢复是安全的。
		if restoreRuninfo != nil {
			restoreRuninfo()
			restoreRuninfo = nil
		}
		report.Render(s)
		path, err := report.WriteJSON(s)
		if err != nil {
			return err
		}
		if path != "" {
			gologger.Info().Msgf("Report saved: %s", path)
		}
		return nil
	}

	workflow, err := audit.Workflow(ctx, factory, render)
	if err != nil {
		return fmt.Errorf("build workflow: %w", err)
	}
	runCtx := audit.WithSeed(ctx, state)
	if traceCtx != nil {
		runCtx = traceCtx(runCtx)
	}
	// 追踪开启后，每个 span 的记录都会被回调打到 stdout，这里拦掉。render 会在
	// report 节点里提前恢复；这里是 workflow 没跑到 report 时的兜底。
	restoreRuninfo = installRuninfoFilter()
	defer func() {
		if restoreRuninfo != nil {
			restoreRuninfo()
		}
	}()

	if _, err := workflow.Invoke(runCtx, map[string]any{}); err != nil {
		return err
	}

	pluginCount, behaviorCount := report.Counts(state)
	gologger.Info().Msgf("Audit finished: %d plugin findings, %d behavioral findings", pluginCount, behaviorCount)
	gologger.Info().Msgf("SKILL scan complete, time %.2f seconds", time.Since(start).Seconds())
	return nil
}
