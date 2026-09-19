package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOptionsSourceAndPresetMutuallyExclusive(t *testing.T) {
	o := &Options{Source: ".", Preset: "claude-code"}
	err := o.validateOptions()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("must reject -s and -p together, got %v", err)
	}
}

func TestValidateOptionsPresetSkipsSkillMdCheck(t *testing.T) {
	// 预设模式下不要求任何路径含 SKILL.md：根目录有没有 skill 由 FindSkills 判定。
	o := &Options{Preset: "claude-code"}
	if err := o.validateOptions(); err != nil {
		t.Fatalf("preset-only options must pass validation: %v", err)
	}
}

func TestValidateOptionsUnknownPreset(t *testing.T) {
	o := &Options{Preset: "no-such-tool"}
	err := o.validateOptions()
	if err == nil || !strings.Contains(err.Error(), "Unknown preset") {
		t.Fatalf("must reject unknown preset, got %v", err)
	}
}

func TestValidateOptionsSourceWithoutSkillMd(t *testing.T) {
	// 父目录批量是合法用法：没有 SKILL.md 不再在参数校验阶段拒绝，形态判定
	// 移到运行时的 skillDirs。
	dir := t.TempDir()
	o := &Options{Source: dir}
	if err := o.validateOptions(); err != nil {
		t.Fatalf("parent directory must pass validation: %v", err)
	}
	o = &Options{Source: filepath.Join(dir, "missing")}
	if err := o.validateOptions(); err == nil {
		t.Fatal("missing path must still fail")
	}
}
