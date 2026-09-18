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
	registerTracing(cfg)

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
	gologger.Info().Msgf("Thread ID: %s", state.TaskID)

	if state.SaveOutput {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}
		gologger.Info().Msgf("Output base directory: %s", outputDir)
	}

	// 缓存查找不受 -no-output 影响：只要上次连同报告一起落过盘，这次就直接渲染，
	// 与参考实现的行为一致。
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

	factory := llm.NewFactory(cfg)
	render := func(s *audit.State) error {
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
	if _, err := workflow.Invoke(runCtx, map[string]any{}); err != nil {
		return err
	}

	pluginCount, behaviorCount := report.Counts(state)
	gologger.Info().Msgf("Audit finished: %d plugin findings, %d behavioral findings", pluginCount, behaviorCount)
	gologger.Info().Msgf("SKILL scan complete, time %.2f seconds", time.Since(start).Seconds())
	return nil
}
