package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/filesystem"
)

// 只读沙箱是唯一真正的执行边界——eino 的工具可见性过滤不是。这里覆盖它的四类
// 拒绝路径。

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

	// root 是 skill 的父目录，恶意 SKILL 放一个指向兄弟目录的软链就能读到 skill
	// 之外；这三条断言就是这个越界的守门人。
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

// glob 的字符类与花括号是上游工具说明向模型承诺的语法。不支持的话模型照写就是
// 静默零结果，而它只会以为目录里没有这类文件——在安全扫描里就是漏报。
func TestGlobSupportsCharClassAndBraces(t *testing.T) {
	cases := []struct {
		pattern string
		match   []string
		noMatch []string
	}{
		{"[abc].md", []string{"a.md", "b.md"}, []string{"d.md", "[abc].md", "x/a.md"}},
		{"*.{ts,tsx}", []string{"a.ts", "a.tsx"}, []string{"a.js", "a.tsx2"}},
		{"{a,b}/*.md", []string{"a/x.md", "b/y.md"}, []string{"c/x.md", "a/x.txt"}},
		{"{a,{b,c}}/*.md", []string{"a/x.md", "b/x.md", "c/x.md"}, []string{"d/x.md"}},
		{"[!abc].md", []string{"d.md"}, []string{"a.md"}},
		{"**/*.md", []string{"x.md", "a/x.md", "a/b/x.md"}, []string{"x.txt"}},
		{"a?c.md", []string{"abc.md"}, []string{"ac.md", "a/c.md"}},
		{"literal[.md", []string{"literal[.md"}, nil},
	}
	for _, c := range cases {
		re, err := GlobToRegexp(c.pattern)
		if err != nil {
			t.Errorf("%q: compile error %v", c.pattern, err)
			continue
		}
		for _, m := range c.match {
			if !re.MatchString(m) {
				t.Errorf("%q must match %q (regexp %s)", c.pattern, m, re)
			}
		}
		for _, m := range c.noMatch {
			if re.MatchString(m) {
				t.Errorf("%q must not match %q (regexp %s)", c.pattern, m, re)
			}
		}
	}
}

// 读长文件时输出封顶，但必须说清楚是被截断的视图，否则模型会把"到此为止"当成
// 文件真的没有更多内容。
func TestReadTruncatesWithNoticeAndOffsetContinues(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(skill, 0o755))
	var sb strings.Builder
	for i := 0; i < 1500; i++ {
		fmt.Fprintf(&sb, "line %d\n", i+1)
	}
	must(t, os.WriteFile(filepath.Join(skill, "big.txt"), []byte(sb.String()), 0o644))
	b := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	c, err := b.Read(ctx, &filesystem.ReadRequest{FilePath: "/myskill/big.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(c.Content, "\n"); n > maxReadLines+1 {
		t.Errorf("output not capped: %d lines", n)
	}
	if !strings.Contains(c.Content, "output truncated") {
		t.Error("truncation must be announced")
	}
	// 续读必须能拿到被截断掉的部分。
	c2, err := b.Read(ctx, &filesystem.ReadRequest{FilePath: "/myskill/big.txt", Offset: 1200})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c2.Content, "line 1200") {
		t.Error("offset must reach past the cap")
	}
}

// grep 的文件枚举不能复用 ls/glob 的 maxListEntries 封顶：超过封顶数量的文件会被
// 静默跳过，"搜过了"伪装成"没搜到"。ls/glob 面向模型照常封顶，grep 必须看全。
func TestGrepSeesPastTheListCap(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(filepath.Join(skill, "zdir"), 0o755))
	// zdir 按路径排序落在 f* 之后，needle 在 GlobInfo 的封顶线之外。
	for i := 0; i < maxListEntries+20; i++ {
		must(t, os.WriteFile(filepath.Join(skill, fmt.Sprintf("f%04d.txt", i)), []byte("plain\n"), 0o644))
	}
	must(t, os.WriteFile(filepath.Join(skill, "zdir", "zneedle.txt"), []byte("NEEDLE here\n"), 0o644))

	b := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	gl, err := b.GlobInfo(ctx, &filesystem.GlobInfoRequest{Path: "/myskill", Pattern: "**/*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(gl) != maxListEntries+1 {
		t.Fatalf("glob must stay capped: got %d entries", len(gl))
	}

	got, err := b.GrepRaw(ctx, &filesystem.GrepRequest{Pattern: "NEEDLE", Path: "/myskill"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "/myskill/zdir/zneedle.txt" {
		t.Fatalf("file past the list cap must be searched, got %+v", got)
	}
}

// grep 先走封顶的 Read 再搜会在长文件上漏报，这里锁定它读全文。
func TestGrepSeesPastTheReadCap(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(skill, 0o755))
	var sb strings.Builder
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&sb, "filler %d\n", i)
	}
	sb.WriteString("NEEDLE after the read cap\n")
	must(t, os.WriteFile(filepath.Join(skill, "big.txt"), []byte(sb.String()), 0o644))

	b := New(root, []string{"/myskill"}, nil)
	got, err := b.GrepRaw(context.Background(), &filesystem.GrepRequest{Pattern: "NEEDLE", Path: "/myskill"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Line != 3001 {
		t.Fatalf("match past the cap must be reported, got %+v", got)
	}
}

// -A/-B 与多行的处理口径：上下文要真的返回，多行要显式报错而不是静默忽略。
func TestGrepContextAndMultiline(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(skill, 0o755))
	must(t, os.WriteFile(filepath.Join(skill, "a.txt"), []byte("before\nHIT\nafter\n"), 0o644))
	b := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	got, err := b.GrepRaw(ctx, &filesystem.GrepRequest{Pattern: "HIT", Path: "/myskill", BeforeLines: 1, AfterLines: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Content, "before") || !strings.Contains(got[0].Content, "after") {
		t.Fatalf("context lines must be returned, got %+v", got)
	}
	if _, err := b.GrepRaw(ctx, &filesystem.GrepRequest{Pattern: "HIT", Path: "/myskill", EnableMultiline: true}); err == nil {
		t.Error("multiline must be rejected explicitly, not silently ignored")
	}
}

// grep 命中过多时必须封顶，并且把"这是被削过的视图"告诉模型——否则"没搜到"会被
// 当成"不存在"。
func TestGrepCapsMatchesAndSaysSo(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(skill, 0o755))
	var sb strings.Builder
	for i := 0; i < maxGrepMatches+50; i++ {
		sb.WriteString("HIT repeated line\n")
	}
	must(t, os.WriteFile(filepath.Join(skill, "a.txt"), []byte(sb.String()), 0o644))

	got, err := New(root, []string{"/myskill"}, nil).GrepRaw(context.Background(),
		&filesystem.GrepRequest{Pattern: "HIT", Path: "/myskill"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > maxGrepMatches+1 {
		t.Fatalf("matches not capped: %d", len(got))
	}
	last := got[len(got)-1]
	if last.Path != "[sandbox]" || !strings.Contains(last.Content, "truncated") {
		t.Fatalf("truncation must be announced, got %+v", last)
	}
}

// ls/glob 的输出同样封顶，避免一次列目录就把上下文预算吃光。
func TestLsAndGlobAreCapped(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	must(t, os.MkdirAll(skill, 0o755))
	for i := 0; i < maxListEntries+20; i++ {
		must(t, os.WriteFile(filepath.Join(skill, fmt.Sprintf("f%04d.txt", i)), []byte("x"), 0o644))
	}
	b := New(root, []string{"/myskill"}, nil)
	ctx := context.Background()

	ls, err := b.LsInfo(ctx, &filesystem.LsInfoRequest{Path: "/myskill"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) > maxListEntries+1 || !strings.Contains(ls[len(ls)-1].Path, "truncated") {
		t.Fatalf("ls must be capped with a notice, got %d entries, last=%+v", len(ls), ls[len(ls)-1])
	}
	gl, err := b.GlobInfo(ctx, &filesystem.GlobInfoRequest{Path: "/myskill", Pattern: "**/*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(gl) > maxListEntries+1 || !strings.Contains(gl[len(gl)-1].Path, "truncated") {
		t.Fatalf("glob must be capped with a notice, got %d entries", len(gl))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
