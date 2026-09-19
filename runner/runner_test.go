package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillDirsSingleSkill(t *testing.T) {
	// 根目录直接含 SKILL.md：按单个 skill 扫，root 为空表示不走确认与汇总。
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Runner{Options: &Options{Source: dir}}
	skills, root, err := r.skillDirs()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" {
		t.Fatalf("single-skill mode must have empty root, got %q", root)
	}
	if len(skills) != 1 || skills[0] != dir {
		t.Fatalf("got %v, want [%s]", skills, dir)
	}
}

func TestSkillDirsParentDirectory(t *testing.T) {
	// 没有 SKILL.md 的目录按父目录批量枚举，与 preset 走同一套清单确认与循环。
	parent := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(parent, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(parent, name, "SKILL.md"), []byte("# s\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := &Runner{Options: &Options{Source: parent}}
	skills, root, err := r.skillDirs()
	if err != nil {
		t.Fatal(err)
	}
	if root != parent {
		t.Fatalf("parent mode must return the root, got %q", root)
	}
	if len(skills) != 2 {
		t.Fatalf("got %v, want both subdirectories", skills)
	}
}

func TestSkillDirsParentWithoutSkills(t *testing.T) {
	// 既不是 skill 也不是父目录：列表为空，由 Run 报 "No skills found under"。
	r := &Runner{Options: &Options{Source: t.TempDir()}}
	skills, root, err := r.skillDirs()
	if err != nil {
		t.Fatal(err)
	}
	if root == "" || len(skills) != 0 {
		t.Fatalf("expected empty list with root set, got root=%q skills=%v", root, skills)
	}
}
