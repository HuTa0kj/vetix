package pluginutils

import (
	"os"
	"path/filepath"
	"testing"
)

// 目录树与哈希必须与 Python 版逐条一致：前者直接进提示词，后者决定缓存目录。
// 这些断言固定在几个已知形状上。
func TestTreeShape(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "SKILL.md"), "# a\n# b\n")
	mustWrite(t, filepath.Join(dir, "data.json"), "{}\n")
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "__pycache__"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "scripts", "run.py"), "print(1)\n")
	// 忽略列表按 basename 匹配，__pycache__ 整体被剪掉。
	mustWrite(t, filepath.Join(dir, "scripts", "__pycache__", "x.pyc"), "junk")

	tree, err := Tree(dir)
	if err != nil {
		t.Fatal(err)
	}

	// 顶部多一层 "."，一级子目录是它的兄弟键而不是嵌套在里面——这是 pstruc 的
	// 输出形状，提示词就是照着它的 repr 写的。
	if _, ok := tree["."]; !ok {
		t.Fatalf("tree must keep the '.' root, got %v", tree)
	}
	if _, ok := tree["scripts"]; !ok {
		t.Fatalf("first-level dirs must be siblings of '.', got %v", tree)
	}
	root := tree["."].(map[string]any)
	if len(root) != 2 {
		t.Fatalf("'.' should hold exactly the root files, got %v", root)
	}
	leaf, ok := root["SKILL.md"].(map[string]any)
	if !ok || leaf["line_count"] != 2 {
		t.Fatalf("file leaf must be {'line_count': N}, got %v", root["SKILL.md"])
	}

	// total_files 是真实文件数：行数增强后的叶子自身算目录、内部计数算文件，
	// 两者在计数上互相抵消。
	top, files, _ := TreeStats(tree)
	if top != 2 {
		t.Errorf("top_level_items = %d, want 2", top)
	}
	if files != 3 {
		t.Errorf("total_files = %d, want 3 (SKILL.md, data.json, scripts/run.py)", files)
	}
}

func TestExtMatchesSplitext(t *testing.T) {
	// Python 的 os.path.splitext 对以点开头的文件名不返回扩展名，所以 .env 会被
	// 判为 rare file，.env.txt 则是 .txt。
	cases := map[string]string{
		".env":     "",
		".env.txt": ".txt",
		"SKILL.md": ".md",
		"README":   "",
		"a.PY":     ".py",
	}
	for in, want := range cases {
		if got := Ext(in); got != want {
			t.Errorf("Ext(%q) = %q, want %q", in, got, want)
		}
	}
	if !IsRiskFile(".env") {
		t.Error(".env must be reported as a rare file")
	}
	if IsRiskFile("notes.md") {
		t.Error("allow-listed extension must not be a rare file")
	}
}

func TestSplitLinesAndNonText(t *testing.T) {
	// Python 的 str.splitlines() 对末尾换行不产生额外空行。
	if n := len(SplitLines("a\nb\n")); n != 2 {
		t.Errorf("trailing newline must not add a line, got %d", n)
	}
	if n := len(SplitLines("a\nb")); n != 2 {
		t.Errorf("unterminated last line still counts, got %d", n)
	}
	if n := len(SplitLines("")); n != 0 {
		t.Errorf("empty content has no lines, got %d", n)
	}

	if ExistNonText([]byte("plain ascii")) {
		t.Error("ascii text must not be flagged")
	}
	if !ExistNonText([]byte{0x01, 0x02, 0x03, 0x04}) {
		t.Error("control bytes must be flagged as non-text")
	}
	if ExistNonText(nil) {
		t.Error("empty content must not be flagged")
	}
}

func TestDirectoryHashIsOrderStable(t *testing.T) {
	// 哈希顺序必须是 os.walk 的遍历顺序（本层文件按名排序，再递归子目录），
	// 而不是全局排序。顺序变了缓存目录就全失效。
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "SKILL.md"), "x\n")
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "a", "b.md"), "y\n")

	first, err := DirectoryHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DirectoryHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("hash must be deterministic: %s != %s", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("expected sha256 hex, got %q", first)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
