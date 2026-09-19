package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// presets 把预设名映射到其 skills 根目录。目录解析延迟到调用时，测试不必捏造
// home；os.UserHomeDir 在 Windows 上取 %USERPROFILE%，Mac/Linux 取 $HOME，
// 三平台适配
var presets = map[string]func() (string, error){
	"claude-code": func() (string, error) {
		return skillsHome(".claude")
	},
	"codex": func() (string, error) {
		return skillsHome(".codex")
	},
}

// skillsHome 拼出 <home>/<agent>/skills，各预设的目录形状一致。
func skillsHome(agent string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, agent, "skills"), nil
}

// resolvePreset 把预设名解析成 skills 根目录。只解析、不校验存在性：root 里
// 有没有 skill 由 FindSkills 枚举时判定。
func resolvePreset(name string) (string, error) {
	build, ok := presets[name]
	if !ok {
		return "", fmt.Errorf("Unknown preset: %s (available: %s)", name, strings.Join(presetNames(), ", "))
	}
	return build()
}

func presetNames() []string {
	names := make([]string, 0, len(presets))
	for n := range presets {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
