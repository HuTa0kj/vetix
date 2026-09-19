package runner

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"

	"vetix/internal/plugin"
	"vetix/internal/report"
)

type Options struct {
	Source      string
	Preset      string
	Language    string
	Output      bool
	NoOutput    bool
	OutputDir   string
	Force       bool
	Debug       bool
	PluginsList bool
	ConfigPath  string
}

func ParseOptions() (*Options, error) {
	options := &Options{}
	flagSet := goflags.NewFlagSet()

	flagSet.CreateGroup("input", "Input",
		flagSet.StringVarP(&options.Source, "source", "s", "", "SKILL directory path or a skills parent directory"),
		flagSet.StringVarP(&options.Preset, "preset", "p", "", "Scan all skills under a preset agent skills root (claude-code, codex)"),
	)

	flagSet.CreateGroup("config", "Config",
		flagSet.StringVarP(&options.ConfigPath, "config", "c", "./config.yaml", "Path to the YAML config file"),
		flagSet.StringVarP(&options.Language, "language", "l", "en", "Output language for audit findings (en, zh)"),
	)

	flagSet.CreateGroup("output", "Output",
		flagSet.BoolVarP(&options.Output, "output", "o", true, "Save the audit report to <output-dir>/<skill-hash-prefix>/report.json"),
		flagSet.BoolVar(&options.NoOutput, "no-output", false, "Disable saving the audit report to a JSON file"),
		flagSet.StringVar(&options.OutputDir, "output-dir", "./output", "Base directory for saved reports"),
		flagSet.BoolVar(&options.Force, "force", false, "Ignore the cached report and force a full re-scan"),
	)

	flagSet.CreateGroup("debug", "Debug",
		flagSet.BoolVarP(&options.Debug, "debug", "d", false, "Enable debug logging"),
	)

	flagSet.CreateGroup("plugin", "Plugin",
		flagSet.BoolVarP(&options.PluginsList, "plugins-list", "pl", false, "List built-in plugins with their names and descriptions"),
	)

	// 不调用 flagSet.Parse()：它每次都会往 ~/.config/<tool>/config.yaml 写一份
	// 自己的配置文件并做配置合并，与我们工作目录下的 config.yaml 语义冲突。
	// 直接解析到导出的 CommandLine，保留 goflags 的 flag 类型与分组写法。
	// 代价是 goflags 的 usageFunc 无法复用（未导出），因此 help 自行渲染。
	flagSet.CommandLine.Usage = func() { usage(os.Stderr) }
	if err := flagSet.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("Parse flags error: %v", err)
	}

	// -pl 是纯查询：打印内置插件清单后即退出，不要求 -s，也不打 banner。
	if options.PluginsList {
		printPlugins(os.Stdout)
		os.Exit(0)
	}

	ShowBanner()

	// gologger 默认 max level 是 Info(3)，而 Warning 是 4，条件为 level <= maxLevel，
	// 所以不显式抬高的话所有 Warning 都会被丢掉——包括"复核结果没解析出来"这类
	// 必须让人看见的降级提示。
	maxLevel := levels.LevelWarning
	if options.Debug {
		maxLevel = levels.LevelDebug
	}
	gologger.DefaultLogger.SetMaxLevel(maxLevel)
	gologger.DefaultLogger.SetTimestampWithFormat(true, levels.LevelFatal, "2006-01-02 15:04:05")

	if err := options.validateOptions(); err != nil {
		return nil, err
	}

	options.outputConfig()

	return options, nil
}

func usage(w *os.File) {
	fmt.Fprintf(w, "%s\n\nUsage:\n  vetix -s <skill-dir> | -p <preset> [flags]\n\nFlags:\n", ToolDesc)
	fmt.Fprint(w, `  INPUT
    -s, -source string   SKILL directory path or a skills parent directory
    -p, -preset string   Scan all skills under a preset agent skills root (claude-code, codex)

  CONFIG
    -c, -config string   Path to the YAML config file (default "./config.yaml")
    -l, -language string Output language for audit findings (en, zh) (default "en")

  OUTPUT
    -o, -output          Save the audit report to <output-dir>/<skill-hash-prefix>/report.json (default true)
    -no-output           Disable saving the audit report to a JSON file
    -output-dir string   Base directory for saved reports (default "./output")
    -force               Ignore the cached report and force a full re-scan

  DEBUG
    -d, -debug           Enable debug logging

  PLUGIN
    -pl, -plugins-list   List built-in plugins with their names and descriptions
`)
}

// printPlugins 输出内置插件清单，样式与报告渲染同源：暗色 ID 作键、粗体名称、
// 描述按终端宽度折行并悬挂缩进到名称列。描述是完整句子，不折行的话窄终端上
// 会顶出屏幕。
func printPlugins(w io.Writer) {
	metas := plugin.List()
	idWidth := 0
	for _, m := range metas {
		if len(m.ID) > idWidth {
			idWidth = len(m.ID)
		}
	}
	descIndent := 2 + idWidth + 2

	fmt.Fprintf(w, "\n%s\n\n", text.FgHiBlue.Sprintf("────────── Built-in Plugins (%d) ──────────", len(metas)))
	for i, m := range metas {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "  %s  %s\n", text.FgHiBlack.Sprintf("%-*s", idWidth, m.ID), text.Bold.Sprint(m.Name))
		lines := report.WrapIndent(m.Description, descIndent)
		fmt.Fprintln(w, strings.Repeat(" ", descIndent)+lines[0])
		for _, l := range lines[1:] {
			fmt.Fprintln(w, l)
		}
	}
	fmt.Fprintln(w)
}

func (o *Options) validateOptions() error {
	if o.Source != "" && o.Preset != "" {
		return fmt.Errorf("-source and -preset are mutually exclusive")
	}
	if o.Source == "" && o.Preset == "" {
		return fmt.Errorf("Source or preset is required")
	}
	// 预设在这里只解析名字：根目录本身不该被要求直接含 SKILL.md，有没有可扫的
	// skill 由 Run 阶段的 FindSkills 枚举时判定。
	if o.Preset != "" {
		if _, err := resolvePreset(o.Preset); err != nil {
			return err
		}
	}
	if o.Source != "" {
		info, err := os.Stat(o.Source)
		if err != nil {
			return fmt.Errorf("Path not found: %s", o.Source)
		}
		if !info.IsDir() {
			return fmt.Errorf("Not a directory: %s", o.Source)
		}
		// 这里不要求 SKILL.md：单 skill 还是父目录批量由 Run 阶段按目录形态
		// 判定，指到父目录上是合法用法。
	}
	if o.Language != "en" && o.Language != "zh" {
		gologger.Warning().Msgf("Unknown language %q, falling back to en", o.Language)
		o.Language = "en"
	}
	return nil
}

func (o *Options) SaveReport() bool {
	return o.Output && !o.NoOutput
}

func (o *Options) outputConfig() {
	if o.Debug {
		gologger.Info().Msgf("Debug mode enabled")
	}
}
