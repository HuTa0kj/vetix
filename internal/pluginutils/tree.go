package pluginutils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"vetix/internal/pytext"
)

// DirectoryHash 是内容寻址的缓存键：每个文件贡献 "相对路径:文件内容 sha256"，
// 用 "\n" 拼接后再做一次 sha256。算法与顺序都不能改，否则同一份 SKILL 会算出
// 不同哈希，既有的缓存目录全部失效。
//
// 顺序不是全局排序：每层目录内分别对子目录名和文件名排序，先输出本层的文件，
// 再按序递归子目录。go 的 filepath.WalkDir 是词法深度优先（目录名排在文件名前
// 时会先进目录），顺序不同，所以这里自己走。
//
// 读取内容时跟随符号链接，但不下钻符号链接目录。
func DirectoryHash(dir string) (string, error) {
	var digests []string
	var walk func(cur, rel string)
	walk = func(cur, rel string) {
		entries, err := os.ReadDir(cur)
		if err != nil {
			return
		}
		var dirs, files []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
				continue
			}
			files = append(files, e.Name())
		}
		sort.Strings(dirs)
		sort.Strings(files)
		for _, name := range files {
			sum, err := fileSHA256(filepath.Join(cur, name))
			if err != nil {
				continue
			}
			p := name
			if rel != "" {
				p = rel + "/" + name
			}
			digests = append(digests, p+":"+sum)
		}
		for _, name := range dirs {
			sub := name
			if rel != "" {
				sub = rel + "/" + name
			}
			walk(filepath.Join(cur, name), sub)
		}
	}
	walk(dir, "")

	h := sha256.Sum256([]byte(strings.Join(digests, "\n")))
	return hex.EncodeToString(h[:]), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 64*1024)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SkipList 是目录树的忽略清单，按 basename 用 fnmatch 匹配：不做路径前缀匹配，
// 也没有 globstar 语义。改动会直接改变喂给模型的目录树文本。
var SkipList = []string{
	"*.log", "*.pyc", "__pycache__", "node_modules", ".env", "dist", "build", "__init__.py",
	"test", "tests", ".git", ".github", "pyproject.toml", "LICENSE", "Dockerfile", ".DS_Store",
	"Thumbs.db", "*.pyo", "*.so", "*.dll", "*.tmp",
}

func skipped(name string) bool {
	for _, pat := range SkipList {
		if ok, err := filepath.Match(pat, name); err == nil && ok {
			return true
		}
	}
	return false
}

// Tree 构造目录树，形状有意保持"别扭"——提示词是在这段文本上调优的：
//   - 顶层根目录的文件挂在键 "." 下；
//   - 每个一级子目录是 "." 的**兄弟**键，而不是嵌套在 "." 里，因为路径按
//     relpath 切分，根目录的 relpath 是 "."，其余目录各成一条键链；
//   - 文件叶子是 {"line_count": N} 而不是文件名对应的 null。
func Tree(skillDir string) (map[string]any, error) {
	out := map[string]any{}
	root := map[string]any{}
	out["."] = root
	if err := buildTree(skillDir, "", out, root); err != nil {
		return nil, err
	}
	return out, nil
}

// buildTree 把 cur 目录的文件写进 node，一级子目录额外挂到 top 上。
func buildTree(cur, rel string, top, node map[string]any) error {
	entries, err := os.ReadDir(cur)
	if err != nil {
		return err
	}
	var dirs, files []string
	for _, e := range entries {
		if skipped(e.Name()) {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e.Name())
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(dirs)
	sort.Strings(files)

	for _, name := range files {
		node[name] = map[string]any{"line_count": countLines(filepath.Join(cur, name))}
	}
	for _, name := range dirs {
		child := map[string]any{}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if rel == "" {
			// 一级子目录在顶层另起一个键。
			top[name] = child
		} else {
			node[name] = child
		}
		if err := buildTree(filepath.Join(cur, name), childRel, top, child); err != nil {
			return err
		}
	}
	return nil
}

// countLines 以文本模式逐行计数，解码失败（例如二进制文件）时记 0。
// 用原始字节数行会得到完全不同的数字。
func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) {
		return 0
	}
	if len(data) == 0 {
		return 0
	}
	n := strings.Count(string(data), "\n")
	if !strings.HasSuffix(string(data), "\n") {
		n++
	}
	return n
}

// TreeStats 统计目录树：map 值计入 total_dirs 并递归，非 map 的叶子计入
// total_files。行数增强后的文件叶子是 map，所以它自身算一个目录、内部的
// line_count 算一个文件，两者恰好抵消，total_files 等于真实文件数。
func TreeStats(tree map[string]any) (topLevel, files, dirs int) {
	var walk func(m map[string]any)
	walk = func(m map[string]any) {
		for _, v := range m {
			if sub, ok := v.(map[string]any); ok {
				dirs++
				walk(sub)
				continue
			}
			files++
		}
	}
	walk(tree)
	return len(tree), files, dirs
}

// TreeRepr 按 Python 字面量风格渲染目录树，因为提示词里的"目录结构"就是这段
// 文本。map 的遍历顺序在 Go 里是随机的，这里按键排序以保证同一输入文本稳定。
// 渲染委托给 pytext：文件名含引号等特殊字符时能得到正确的转义，而不是拼出
// 破损的字面量。
func TreeRepr(tree map[string]any) string {
	return pytext.Value(tree)
}
