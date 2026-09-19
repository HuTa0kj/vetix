package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/projectdiscovery/gologger"

	"vetix/internal/audit"
	"vetix/internal/config"
	"vetix/internal/llm"
	"vetix/internal/plugin"
	"vetix/internal/pluginutils"
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

	cfg, err := config.Load(r.Options.ConfigPath)
	if err != nil {
		return err
	}

	outputDir, err := filepath.Abs(r.Options.OutputDir)
	if err != nil {
		return err
	}

	if r.Options.SaveReport() {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}
		gologger.Info().Msgf("Output base directory: %s", outputDir)
	}

	skills, root, err := r.skillDirs()
	if err != nil {
		return err
	}
	if root != "" {
		if len(skills) == 0 {
			// 枚举只看直接子目录，指到嵌套两层的容器目录会一无所获，报错里
			// 说清规则，用户才知道该往深处指一层或换 -p。
			return fmt.Errorf("No skills found under: %s (a skill is a subdirectory containing SKILL.md, direct subdirectories only)", root)
		}
		confirmed, err := confirmScan(os.Stdout, os.Stdin, root, skills)
		if err != nil {
			return err
		}
		if !confirmed {
			gologger.Info().Msg("Scan cancelled")
			return nil
		}
	}

	results := make([]skillResult, 0, len(skills))
	var scanErrs []error
	for i, dir := range skills {
		if len(skills) > 1 {
			gologger.Info().Msgf("Scanning skill %d/%d: %s", i+1, len(skills), filepath.Base(dir))
		}
		res, err := r.scanSkill(ctx, cfg, dir, outputDir)
		if err != nil {
			// 批量场景下单个 skill 失败（模型配置错、目录损坏）不该吞掉其余
			// skill 的结果，继续扫，结尾在汇总表和错误码里一并收账。
			gologger.Warning().Msgf("Scan failed: %s: %v", filepath.Base(dir), err)
			scanErrs = append(scanErrs, fmt.Errorf("%s: %w", filepath.Base(dir), err))
			results = append(results, skillResult{Name: filepath.Base(dir), Failed: true})
			continue
		}
		results = append(results, res)
	}

	if root != "" {
		printSummary(os.Stdout, results)
	}
	return errors.Join(scanErrs...)
}

// skillDirs 返回待扫描的 skill 目录列表（绝对路径）与批量模式的根目录；单 skill
// 模式下 root 为空。单 skill 是列表的退化情形，后续缓存检查与 workflow 调用共用
// 同一份代码。
func (r *Runner) skillDirs() ([]string, string, error) {
	if r.Options.Preset != "" {
		root, err := resolvePreset(r.Options.Preset)
		if err != nil {
			return nil, "", err
		}
		skills, err := pluginutils.FindSkills(root)
		if err != nil {
			return nil, "", fmt.Errorf("enumerate skills in %s: %w", root, err)
		}
		return skills, root, nil
	}
	source, err := filepath.Abs(r.Options.Source)
	if err != nil {
		return nil, "", err
	}
	// 直接含 SKILL.md 就是单个 skill；否则按父目录枚举直接子目录。两种形态共用
	// 同一套枚举、确认与循环，用户不需要记第二个入口；指错了目录也会先列出清单
	// 等 y/n，而不是静默扫一堆无关内容。
	if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err == nil {
		return []string{source}, "", nil
	}
	skills, err := pluginutils.FindSkills(source)
	if err != nil {
		return nil, "", fmt.Errorf("enumerate skills in %s: %w", source, err)
	}
	return skills, source, nil
}

func (r *Runner) scanSkill(ctx context.Context, cfg *config.Config, skillDir, outputDir string) (skillResult, error) {
	start := time.Now()

	state := &audit.State{
		TaskID:     newTaskID(),
		SkillDir:   skillDir,
		Workspace:  filepath.Dir(skillDir),
		OutputDir:  outputDir,
		SaveOutput: r.Options.SaveReport(),
		DetectedAt: time.Now().Format("2006-01-02T15:04:05-07:00"),
		Language:   r.Options.Language,
	}
	// Thread ID 同时是 LangSmith 的根 span id。
	gologger.Info().Msgf("Thread ID: %s", state.TaskID)

	// 缓存查找不受 -no-output 影响：只要上次连同报告一起落过盘，这次就直接渲染。
	if !r.Options.Force {
		cached, err := audit.LoadCachedReport(skillDir, outputDir)
		if err != nil {
			return skillResult{}, err
		}
		if cached != nil {
			cached.SaveOutput = state.SaveOutput
			report.Render(cached)
			gologger.Info().Msg("Loaded cached report (use -force to re-scan)")
			return newSkillResult(cached, true), nil
		}
	}

	// 缓存命中时 pipeline 不跑，没有 span 可发，所以 tracing 在缓存未命中后才注册。
	handler, traceCtx := registerTracing(cfg, state.TaskID)

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
		return skillResult{}, fmt.Errorf("build workflow: %w", err)
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

	var invokeOpts []compose.Option
	if handler != nil {
		invokeOpts = append(invokeOpts, compose.WithCallbacks(handler))
	}
	if _, err := workflow.Invoke(runCtx, map[string]any{}, invokeOpts...); err != nil {
		return skillResult{}, err
	}

	pluginCount, behaviorCount := report.Counts(state)
	gologger.Info().Msgf("Audit finished: %d plugin findings, %d behavioral findings", pluginCount, behaviorCount)
	gologger.Info().Msgf("SKILL scan complete, time %.2f seconds", time.Since(start).Seconds())
	return newSkillResult(state, false), nil
}
