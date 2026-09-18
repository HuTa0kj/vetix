package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk/filesystem"
)

// ErrReadOnly 是 agent 挂载的 write_file / edit_file 的失败口径：必须稳定报错，
// 不能静默成功。
var ErrReadOnly = errors.New("read-only backend: write operations are not permitted")

// New 构造只读沙箱。root 是 skill 的父目录（workspace），allow 是允许读取的
// 虚拟路径前缀（如 "/my-skill"）；skillFS 提供 /skills/** 下的内嵌 helper skill。
//
// 权限必须在这里强制，因为上游框架的"权限"都不是执行边界：eino 的 ToolInfos
// 过滤只影响模型可见的工具列表，模型幻觉调用被隐藏的工具时 dispatch 仍按
// ToolsNodeConfig.Tools 查名字并执行。
func New(root string, allow []string, skillFS fs.FS) *Backend {
	return &Backend{root: root, allow: allow, skillFS: skillFS}
}

type Backend struct {
	root    string
	allow   []string
	skillFS fs.FS
}

const skillsPrefix = "/skills"

func (b *Backend) virtual(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if !strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("path must be absolute: %q", p)
	}
	clean := path.Clean(p)
	if clean == "." {
		clean = "/"
	}
	// 用 path.Clean 之后的前缀判断而非子串判断：否则文件名里含 ".." 的合法路径
	// 会被误拒，而真正的向上穿越又拦不住。
	if clean == "/.." || strings.HasPrefix(clean, "/../") {
		return "", fmt.Errorf("path traversal not allowed: %q", p)
	}
	return clean, nil
}

func (b *Backend) allowed(v string) bool {
	if v == skillsPrefix || strings.HasPrefix(v, skillsPrefix+"/") {
		return b.skillFS != nil
	}
	if v == "/" {
		return true
	}
	for _, a := range b.allow {
		if v == a || strings.HasPrefix(v, a+"/") {
			return true
		}
	}
	return false
}

func (b *Backend) resolve(ctx context.Context, p string) (string, error) {
	v, err := b.virtual(p)
	if err != nil {
		return "", err
	}
	if !b.allowed(v) {
		return "", fmt.Errorf("path not permitted: %q", p)
	}
	full := filepath.Join(b.root, filepath.FromSlash(strings.TrimPrefix(v, "/")))
	if err := b.noSymlink(full); err != nil {
		return "", err
	}
	return full, nil
}

// noSymlink 从 root 起逐段 Lstat，拒绝路径中任意一段是符号链接。root 是 skill
// 的父目录，恶意 SKILL 只要放一个指向兄弟目录的软链就能读到 skill 之外的内容。
func (b *Backend) noSymlink(full string) error {
	rel, err := filepath.Rel(b.root, full)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path escapes root: %q", full)
	}
	cur := b.root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		cur = filepath.Join(cur, seg)
		fi, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not allowed in path: %q", cur)
		}
	}
	return nil
}

// embedded 把 /skills/... 映射到内嵌 FS。
func (b *Backend) embedded(v string) (string, bool) {
	if b.skillFS == nil || !strings.HasPrefix(v, skillsPrefix) {
		return "", false
	}
	return "skills" + strings.TrimPrefix(v, skillsPrefix), true
}

func (b *Backend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	v, err := b.virtual(req.FilePath)
	if err != nil {
		return nil, err
	}
	var raw []byte
	if ep, ok := b.embedded(v); ok {
		if !b.allowed(v) {
			return nil, fmt.Errorf("path not permitted: %q", req.FilePath)
		}
		raw, err = fs.ReadFile(b.skillFS, ep)
	} else {
		full, rerr := b.resolve(ctx, req.FilePath)
		if rerr != nil {
			return nil, rerr
		}
		raw, err = os.ReadFile(full)
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	off := req.Offset
	if off < 1 {
		off = 1
	}
	if off > len(lines) {
		return &filesystem.FileContent{Content: ""}, nil
	}
	sel := lines[off-1:]
	if req.Limit > 0 && req.Limit < len(sel) {
		sel = sel[:req.Limit]
	}
	return &filesystem.FileContent{Content: strings.Join(sel, "\n")}, nil
}

func (b *Backend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	v, err := b.virtual(req.Path)
	if err != nil {
		return nil, err
	}
	if ep, ok := b.embedded(v); ok {
		if !b.allowed(v) {
			return nil, fmt.Errorf("path not permitted: %q", req.Path)
		}
		return b.lsEmbedded(v, ep)
	}
	full, err := b.resolve(ctx, req.Path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	var out []filesystem.FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		// 软链一律不上报，避免模型顺着它去读 root 之外的内容。
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		child := path.Join(v, e.Name())
		if !b.allowed(child) {
			continue
		}
		out = append(out, filesystem.FileInfo{
			Path:       child,
			IsDir:      e.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (b *Backend) lsEmbedded(v, ep string) ([]filesystem.FileInfo, error) {
	entries, err := fs.ReadDir(b.skillFS, ep)
	if err != nil {
		return nil, err
	}
	var out []filesystem.FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, filesystem.FileInfo{
			Path:       path.Join(v, e.Name()),
			IsDir:      e.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (b *Backend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	base := req.Path
	if base == "" {
		base = "/"
	}
	v, err := b.virtual(base)
	if err != nil {
		return nil, err
	}
	if !b.allowed(v) {
		return nil, fmt.Errorf("path not permitted: %q", req.Path)
	}
	re, err := GlobToRegexp(req.Pattern)
	if err != nil {
		return nil, err
	}
	if ep, ok := b.embedded(v); ok {
		return b.globEmbedded(v, ep, re)
	}
	full, err := b.resolve(ctx, base)
	if err != nil {
		return nil, err
	}
	var out []filesystem.FileInfo
	err = filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != full && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(full, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !re.MatchString(rel) {
			return nil
		}
		child := path.Join(v, rel)
		if !b.allowed(child) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, filesystem.FileInfo{
			Path:       child,
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (b *Backend) globEmbedded(v, ep string, re *regexp.Regexp) ([]filesystem.FileInfo, error) {
	var out []filesystem.FileInfo
	err := fs.WalkDir(b.skillFS, ep, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, ep), "/")
		if !re.MatchString(rel) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, filesystem.FileInfo{
			Path:       path.Join(v, rel),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (b *Backend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	base := req.Path
	if base == "" {
		base = "/"
	}
	files, err := b.GlobInfo(ctx, &filesystem.GlobInfoRequest{Pattern: "**/*", Path: base})
	if err != nil {
		return nil, err
	}
	// 提示词教模型用 "p1|p2|p3" 批量搜索，RE2 支持；lookaround / backreference
	// 不支持，这里把原因写进错误里让模型自己改正则。
	pat := req.Pattern
	if req.CaseInsensitive {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp %q: %w (Go RE2 syntax; lookaround, backreferences and \\1 are not supported)", req.Pattern, err)
	}
	var globRe *regexp.Regexp
	if req.Glob != "" {
		globRe, err = GlobToRegexp(req.Glob)
		if err != nil {
			return nil, err
		}
	}
	fileType := strings.ToLower(strings.TrimPrefix(req.FileType, "."))
	var out []filesystem.GrepMatch
	for _, f := range files {
		if f.IsDir {
			continue
		}
		if globRe != nil && !globRe.MatchString(path.Base(f.Path)) && !globRe.MatchString(f.Path) {
			continue
		}
		if fileType != "" && strings.TrimPrefix(path.Ext(f.Path), ".") != fileType {
			continue
		}
		c, err := b.Read(ctx, &filesystem.ReadRequest{FilePath: f.Path})
		if err != nil {
			continue
		}
		for i, line := range strings.Split(c.Content, "\n") {
			if re.MatchString(line) {
				out = append(out, filesystem.GrepMatch{Content: line, Path: f.Path, Line: i + 1})
			}
		}
	}
	return out, nil
}

func (b *Backend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	return ErrReadOnly
}

func (b *Backend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	return ErrReadOnly
}

// GlobToRegexp 把 glob 转成 RE2 正则：** 跨目录、* 不跨、? 匹配单字符。
// "**/" 必须能匹配零层目录，否则 "**/*.md" 会漏掉根目录下的文件。
func GlobToRegexp(pattern string) (*regexp.Regexp, error) {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			switch {
			case i+1 < len(pattern) && pattern[i+1] == '*':
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					sb.WriteString("(?:.*/)?")
					i += 2
				} else {
					sb.WriteString(".*")
					i++
				}
			default:
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteString("$")
	return regexp.Compile(sb.String())
}
