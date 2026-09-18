package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/filesystem"

	"vetix/internal/assets"
	"vetix/internal/sandbox"
)

// helper skill 的路径同时出现在提示词与沙箱白名单里，两边任何一处改动都可能让模型
// 读到一个"路径不允许"。这里把提示词里给出的那个路径拿去真读一次，锁住这个一致性。
func TestHelperSkillIsReachableFromTheSandbox(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# s\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 与 behavioralAgent 传参一致：workspace 为根，允许 skill 目录与 helper skill 目录。
	b := sandbox.New(root, []string{"/myskill", helperSkillDir}, assets.SkillsFS())

	c, err := b.Read(context.Background(), &filesystem.ReadRequest{FilePath: HelperSkillPath})
	if err != nil {
		t.Fatalf("the path advertised in BehavioralPrompt must be readable: %v", err)
	}
	if !strings.Contains(c.Content, "Skill Security Scanner") {
		t.Fatalf("unexpected helper skill content: %q", c.Content[:min(80, len(c.Content))])
	}

	// 提示词里点名了这个路径，否则模型没有途径知道它存在。
	if !strings.Contains(BehavioralPrompt(snapshot{SkillDir: skill, SkillName: "myskill", Language: "en"}), HelperSkillPath) {
		t.Error("BehavioralPrompt must advertise the helper skill path")
	}

	// 提示词里给模型的目标目录必须是沙箱虚拟路径（/<SkillName>），并真的可读：
	// 给宿主绝对路径模型会照着 read_file，然后整轮浪费在 path not permitted 上。
	prompt := BehavioralPrompt(snapshot{SkillDir: skill, SkillName: "myskill", Language: "en"})
	if !strings.Contains(prompt, "SKILL directory path: /myskill") {
		t.Errorf("BehavioralPrompt must advertise the sandbox virtual path of the target skill, got: %q", prompt)
	}
	if _, err := b.Read(context.Background(), &filesystem.ReadRequest{FilePath: "/myskill/SKILL.md"}); err != nil {
		t.Fatalf("the advertised target skill path must be readable: %v", err)
	}
}

// 挂载了内嵌 skillFS 不等于整棵 /skills 树对外开放，白名单仍然说了算。
func TestEmbeddedFSIsStillGatedByAllowList(t *testing.T) {
	b := sandbox.New(t.TempDir(), []string{helperSkillDir}, assets.SkillsFS())
	ctx := context.Background()

	if _, err := b.Read(ctx, &filesystem.ReadRequest{FilePath: HelperSkillPath}); err != nil {
		t.Fatalf("allowed path must be readable: %v", err)
	}
	if _, err := b.Read(ctx, &filesystem.ReadRequest{FilePath: "/skills/something-else/SKILL.md"}); err == nil {
		t.Error("a path outside the allow list must be rejected even when skillFS is mounted")
	}
}
