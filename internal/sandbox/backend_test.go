package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/filesystem"
)

// 只读沙箱是唯一真正的执行边界——eino 的工具可见性过滤和 Python 版 deepagents 的
// FilesystemPermission（未匹配时默认 allow）都不是。这里覆盖它的四类拒绝路径。

func TestReadOnlyAndAllowList(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	sibling := filepath.Join(root, "sibling")
	must(t, os.MkdirAll(skill, 0o755))
	must(t, os.MkdirAll(sibling, 0o755))
	must(t, os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# a\n# b\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(sibling, "secret.txt"), []byte("secret"), 0o644))

	sb := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	c, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "/myskill/SKILL.md"})
	if err != nil || c == nil || c.Content == "" {
		t.Fatalf("read allowed path: content=%v err=%v", c, err)
	}

	if err := sb.Write(ctx, &filesystem.WriteRequest{FilePath: "/myskill/x.txt", Content: "x"}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("write must be rejected, got %v", err)
	}
	if err := sb.Edit(ctx, &filesystem.EditRequest{FilePath: "/myskill/SKILL.md", OldString: "a", NewString: "b"}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("edit must be rejected, got %v", err)
	}

	if _, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "/sibling/secret.txt"}); err == nil {
		t.Error("read outside allow-list must be rejected")
	}
	if _, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "myskill/SKILL.md"}); err == nil {
		t.Error("relative path must be rejected")
	}
	if _, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "/../sibling/secret.txt"}); err == nil {
		t.Error("'..' traversal must be rejected")
	}

	if ls, err := sb.LsInfo(ctx, &filesystem.LsInfoRequest{Path: "/myskill"}); err != nil || len(ls) == 0 {
		t.Errorf("LsInfo on allowed dir: %v %v", ls, err)
	}
	if g, err := sb.GrepRaw(ctx, &filesystem.GrepRequest{Pattern: "a|b", Path: "/myskill"}); err != nil || len(g) == 0 {
		t.Errorf("GrepRaw with alternation: %v %v", g, err)
	}
	// 提示词教模型用 RE2 之外的语法时要给出可自纠的错误。
	if _, err := sb.GrepRaw(ctx, &filesystem.GrepRequest{Pattern: "(?<=a)b", Path: "/myskill"}); err == nil || !strings.Contains(err.Error(), "RE2") {
		t.Errorf("lookaround should fail with an actionable error, got %v", err)
	}
	if gl, err := sb.GlobInfo(ctx, &filesystem.GlobInfoRequest{Pattern: "**/*.md", Path: "/myskill"}); err != nil || len(gl) == 0 {
		t.Errorf("GlobInfo **/ must match root-level files: %v %v", gl, err)
	}
}

func TestSymlinksAreRejected(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	sibling := filepath.Join(root, "sibling")
	must(t, os.MkdirAll(skill, 0o755))
	must(t, os.MkdirAll(sibling, 0o755))
	must(t, os.WriteFile(filepath.Join(sibling, "secret.txt"), []byte("secret"), 0o644))
	must(t, os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("hello"), 0o644))
	must(t, os.Symlink(filepath.Join(sibling, "secret.txt"), filepath.Join(skill, "link.txt")))
	must(t, os.Symlink(sibling, filepath.Join(skill, "linkdir")))

	sb := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	// Python 版的 root 是 skill 的父目录且会 resolve 符号链接，恶意 SKILL 放一个
	// 指向兄弟目录的软链就能读到 skill 之外；这三条断言就是这个越界的守门人。
	if _, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "/myskill/link.txt"}); err == nil {
		t.Error("reading a symlinked file must be rejected")
	}
	if _, err := sb.Read(ctx, &filesystem.ReadRequest{FilePath: "/myskill/linkdir/secret.txt"}); err == nil {
		t.Error("traversing a symlinked dir must be rejected")
	}
	ls, err := sb.LsInfo(ctx, &filesystem.LsInfoRequest{Path: "/myskill"})
	if err != nil {
		t.Fatalf("LsInfo: %v", err)
	}
	for _, e := range ls {
		if strings.Contains(e.Path, "link") {
			t.Errorf("symlinks must not be listed, leaked %s", e.Path)
		}
	}
}

func TestEmbeddedSkillsFS(t *testing.T) {
	sb := New(t.TempDir(), nil, os.DirFS("."))
	if _, err := sb.Read(context.Background(), &filesystem.ReadRequest{FilePath: "/skills/x"}); err == nil {
		t.Error("reading embedded skills without an allow entry must be rejected")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
