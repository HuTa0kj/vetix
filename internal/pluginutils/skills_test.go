package pluginutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSkills(t *testing.T) {
	dir := t.TempDir()
	// 两个合法 skill。
	for _, name := range []string{"beta", "alpha"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(dir, "beta", "SKILL.md"), "# beta\n")
	mustWrite(t, filepath.Join(dir, "alpha", "SKILL.md"), "# alpha\n")
	// 无 SKILL.md 的子目录、普通文件、隐藏目录都不算 skill。
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "notes.txt"), "not a skill\n")
	if err := os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, ".hidden", "SKILL.md"), "# hidden\n")
	// symlink 目录指向一个合法 skill，但枚举阶段就必须被跳过。
	if err := os.Symlink(filepath.Join(dir, "alpha"), filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	// SKILL.md 本身是 symlink 的目录同样跳过：沙箱稍后也会拒绝它。
	if err := os.MkdirAll(filepath.Join(dir, "linked-md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "beta", "SKILL.md"), filepath.Join(dir, "linked-md", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	skills, err := FindSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	// 结果按目录名排序，路径都是 root 下的直接子目录。
	want := []string{filepath.Join(dir, "alpha"), filepath.Join(dir, "beta")}
	if len(skills) != len(want) {
		t.Fatalf("got %v, want %v", skills, want)
	}
	for i, s := range skills {
		if s != want[i] {
			t.Errorf("skills[%d] = %s, want %s", i, s, want[i])
		}
	}
}

func TestFindSkillsEmptyRoot(t *testing.T) {
	// 没有 skill 返回空切片而不是错误：要不要报错由调用方决定。
	skills, err := FindSkills(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Fatalf("expected no skills, got %v", skills)
	}
}
