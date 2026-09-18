package audit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/projectdiscovery/gologger"

	"vetix/internal/plugin"
	"vetix/internal/pluginutils"
)

// GatherBaseInfo 收集 SKILL 的基本信息：名称、目录树（含每个文件的行数）、
// 内容哈希，并判定是否只有一个 SKILL.md。
func GatherBaseInfo(h *stateHandle) error {
	snap, err := h.snapshot()
	if err != nil {
		return err
	}
	dir := snap.SkillDir

	name, err := skillName(dir)
	if err != nil {
		return err
	}
	gologger.Info().Msgf("Start scan SKILL: %s", name)

	content, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return fmt.Errorf("read SKILL.md: %w", err)
	}

	tree, err := pluginutils.Tree(dir)
	if err != nil {
		return fmt.Errorf("build skill tree: %w", err)
	}
	top, files, dirs := pluginutils.TreeStats(tree)

	hash, err := pluginutils.DirectoryHash(dir)
	if err != nil {
		return fmt.Errorf("compute directory hash: %w", err)
	}
	gologger.Debug().Msgf("SKILL hash: %s, tree stats: top_level=%d files=%d dirs=%d",
		hash, top, files, dirs)

	single := isSingleSkillFile(dir)
	if single {
		gologger.Info().Msg("Found 1 file in the SKILL directory")
	} else {
		gologger.Info().Msgf("Found %d files in the SKILL directory", files)
	}

	return h.with(func(s *State) {
		s.SkillName = name
		s.SkillContent = string(content)
		s.Tree = tree
		s.Stats = TreeStats{TopLevel: top, Files: files, Dirs: dirs}
		s.DirectoryHash = hash
		s.SingleSkill = single
	})
}

// skillName 取第一个含 SKILL.md 的目录的 basename。遍历是自顶向下的，
// 所以正常情况下就是最外层目录。
func skillName(dir string) (string, error) {
	found := ""
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if _, serr := os.Stat(filepath.Join(p, "SKILL.md")); serr == nil {
			found = filepath.Base(p)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("SKILL.md not found")
	}
	return found, nil
}

// isSingleSkillFile 判定目录下是否只有 SKILL.md 一个条目。注意隐藏文件也计入，
// 所以 SKILL.md + .DS_Store 会被判为多文件目录——这条边界别"顺手修掉"，
// 否则单文件快速路径的触发条件会变。
func isSingleSkillFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	if len(entries) != 1 {
		return false
	}
	return entries[0].Name() == "SKILL.md"
}

// PluginCheck 对所有文件跑插件。插件级异常在 plugin 包内被隔离成警告。
func PluginCheck(h *stateHandle) error {
	snap, err := h.snapshot()
	if err != nil {
		return err
	}

	issues, err := plugin.ScanDirectory(snap.SkillDir)
	if err != nil {
		return err
	}

	total := 0
	for _, list := range issues {
		total += len(list)
	}
	gologger.Info().Msgf("Plugin check revealed %d security risks", total)

	return h.with(func(s *State) { s.PluginsCheckFindings = issues })
}
