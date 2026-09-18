package pluginutils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// DirectoryHash 是内容寻址的缓存键：每个文件贡献 "相对路径:文件内容 sha256"，
// 用 "\n" 拼接后再做一次 sha256。算法与顺序都必须和 Python 版逐字一致，
// 否则同一份 SKILL 会算出不同哈希，既有的缓存目录直接失效。
//
// 顺序不是全局排序，而是 os.walk 的遍历顺序：每层目录内分别对子目录名和文件名
// 排序，先输出本层的文件，再按序递归子目录。go 的 filepath.WalkDir 是词法深度
// 优先（目录名排在文件名前时会先进目录），顺序不同，所以这里自己走。
//
// 读取内容时跟随符号链接（与 Python 的 open 一致），但不下钻符号链接目录。
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

// SkipList 与 Python 版传给 pstruc 的 to_ignore 完全一致。
//
// pstruc 用 fnmatch 只匹配 basename，不做路径前缀匹配，也没有 globstar 语义；
// 这里用同一套规则，否则目录树内容会与 Python 版不同，进而改变喂给模型的文本。
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

// Tree 复刻 pstruc output_format="dict" 加行数增强后的形状。
//
// 形状本身很别扭但必须保留——提示词是在它的 Python repr 上调优过的：
//   - 顶层根目录的文件挂在键 "." 下；
//   - 每个一级子目录是 "." 的**兄弟**键，而不是嵌套在 "." 里，因为 pstruc 用
//     os.path.relpath 切分路径，根目录的 relpath 是 "."，其余目录各成一条键链；
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

// buildTree 把 cur 目录的文件写进 node，子目录按 pstruc 的规则挂到 top 上。
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

// countLines 与 Python 的 _enrich_tree_with_line_counts 一致：以文本模式打开并逐行
// 计数，解码失败（例如二进制文件）时记 0。用原始字节数行会得到完全不同的数字。
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

// TreeStats 复刻 Python 的 get_tree_stats：map 值计入 total_dirs 并递归，非 map
// 的叶子计入 total_files。行数增强后的文件叶子是 map，所以它自身算一个目录、
// 内部的 line_count 算一个文件，两者恰好抵消，total_files 等于真实文件数。
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

// TreeRepr 渲染成 Python dict 的 repr，因为提示词里的"目录结构"就是这段文本。
// Python 的 dict 顺序取决于写入顺序，这里按键排序以保证同一输入文本稳定。
func TreeRepr(tree map[string]any) string {
	var sb strings.Builder
	writePy(&sb, tree)
	return sb.String()
}

func writePy(sb *strings.Builder, tree map[string]any) {
	sb.WriteString("{")
	keys := make([]string, 0, len(tree))
	for k := range tree {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("'")
		sb.WriteString(k)
		sb.WriteString("': ")
		switch v := tree[k].(type) {
		case map[string]any:
			writePy(sb, v)
		case int:
			sb.WriteString(fmt.Sprintf("%d", v))
		case nil:
			sb.WriteString("None")
		default:
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}
	sb.WriteString("}")
}
