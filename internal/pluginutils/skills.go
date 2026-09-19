package pluginutils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FindSkills 枚举 root 下含 SKILL.md 的直接子目录，即一个 preset 根目录里的全部
// skill。symlink 一律跳过——全库对 symlink 的口径是拒绝，列出来的 skill 即便指向
// 内部可用目录，沙箱阶段也会拒绝读取，不如在枚举时不把它算作 skill；`.` 开头的
// 目录是工具杂物（.git、缓存等）的常见藏身处，同样不算。
func FindSkills(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var skills []string
	for _, e := range entries {
		// ReadDir 基于 lstat，symlink 目录在这里就不会被 IsDir 放行。
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(root, e.Name())
		info, err := os.Lstat(filepath.Join(dir, "SKILL.md"))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		skills = append(skills, dir)
	}
	sort.Strings(skills)
	return skills, nil
}
