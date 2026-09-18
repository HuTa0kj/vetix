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
	"unicode/utf8"

	"github.com/cloudwego/eino/adk/filesystem"
)

// ErrReadOnly 是 agent 挂载的 write_file / edit_file 的失败口径：必须稳定报错，
// 不能静默成功。
var ErrReadOnly = errors.New("read-only backend: write operations are not permitted")

// 单次读/搜的输出上限。给输出封顶不是可选的优化，它同时解决两个问题：
//
//  1. deep agent 的大结果 offload 在这里是条死路。上游 middleware 默认在结果超过
//     约 80KB 时调用 Backend.Write 落盘，而只读 backend 恒返回 ErrReadOnly，于是
//     工具整体失败——模型连已经读到的部分都拿不到，只能反复重试同一个文件。
//     deep.Config 不暴露关掉它的开关，所以在读侧削峰是唯一守得住的边界。
//  2. 上下文预算本来是按 50 次模型调用设计的，一次全量读回灌就能吃掉大半。
//
// 超限时附一行说明，让模型知道自己看到的是被截断的视图、可以换参数继续——静默截断
// 会被误读成"文件/目录到此为止"，在安全扫描里就是漏报。
const (
	maxReadLines   = 1000
	maxReadBytes   = 60 * 1024
	maxListEntries = 200
	maxGrepMatches = 200
	maxGrepBytes   = 48 * 1024
	maxGrepLine    = 4 * 1024
)

// notice 以一条"路径"的形式搭在 ls/glob 的返回值后面。FileInfo 没有承载提示的字段，
// 而上游只是把 Path 逐行拼起来输出，所以这是唯一能插话的位置。
func notice(format string, args ...any) filesystem.FileInfo {
	return filesystem.FileInfo{Path: "[sandbox] " + fmt.Sprintf(format, args...)}
}

// withNotice 在封顶后的列表末尾追加提示。切片表达式带上第三个参数，强制 append 另起
// 底层数组，免得写进被截掉的那部分元素里。
func withNotice(items []filesystem.FileInfo, format string, args ...any) []filesystem.FileInfo {
	return append(items[:len(items):len(items)], notice(format, args...))
}

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
	if v == "/" {
		return true
	}
	for _, a := range b.allow {
		if v == a || strings.HasPrefix(v, a+"/") {
			// /skills/** 的内容来自内嵌 FS：没挂载 skillFS 时即使列进白名单也无
			// 可读。反过来说，白名单没有列到的 /skills 路径一律拒绝——挂载了
			// skillFS 不等于整棵内嵌树都对外开放，否则 allow 就是个摆设。
			if (v == skillsPrefix || strings.HasPrefix(v, skillsPrefix+"/")) && b.skillFS == nil {
				return false
			}
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
		return "", b.notPermitted(p)
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

// notPermitted 在错误里带上白名单。提示词给不了模型所有路径知识（helper skill、
// 虚拟挂载点），盲猜格式要花模型整轮工具调用；列出允许路径让它一次自纠。白名单
// 是虚拟路径，不含宿主信息。
func (b *Backend) notPermitted(p string) error {
	return fmt.Errorf("path not permitted: %q (permitted paths: %s)", p, strings.Join(b.allow, ", "))
}

func (b *Backend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	text, err := b.readAll(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(text, "\n")
	off := req.Offset
	if off < 1 {
		off = 1
	}
	if off > len(lines) {
		return &filesystem.FileContent{Content: ""}, nil
	}
	total := len(lines)
	sel := lines[off-1:]
	if req.Limit > 0 && req.Limit < len(sel) {
		sel = sel[:req.Limit]
	}
	last := off - 1 + len(sel)
	// capped 只标记沙箱自己削过的部分。模型主动传的 limit 不算截断——它知道自己要了
	// 多少行，只需告诉它后面还有。
	capped := false
	if len(sel) > maxReadLines {
		sel = sel[:maxReadLines]
		last = off - 1 + maxReadLines
		capped = true
	}
	content := strings.Join(sel, "\n")
	if len(content) > maxReadBytes {
		content = truncateBytes(content, maxReadBytes)
		capped = true
	}
	if capped {
		content += fmt.Sprintf("\n[output truncated: showing lines %d-%d of %d; re-read with offset to continue]", off, last, total)
	} else if last < total {
		content += fmt.Sprintf("\n[%d more lines available (total %d); re-read with offset to continue]", total-last, total)
	}
	return &filesystem.FileContent{Content: content}, nil
}

// readAll 读整个文件，不做封顶。Read 在它上面做窗口与削峰；GrepRaw 必须拿到全文，
// 否则超过 maxReadLines 的部分搜不到——那是漏报，不是省预算。
func (b *Backend) readAll(ctx context.Context, p string) (string, error) {
	v, err := b.virtual(p)
	if err != nil {
		return "", err
	}
	var raw []byte
	if ep, ok := b.embedded(v); ok {
		if !b.allowed(v) {
			return "", b.notPermitted(p)
		}
		raw, err = fs.ReadFile(b.skillFS, ep)
	} else {
		full, rerr := b.resolve(ctx, p)
		if rerr != nil {
			return "", rerr
		}
		raw, err = os.ReadFile(full)
	}
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n"), nil
}

// truncateBytes 按字节截断，但不切开一个 UTF-8 编码点。
func truncateBytes(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

func (b *Backend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	v, err := b.virtual(req.Path)
	if err != nil {
		return nil, err
	}
	if ep, ok := b.embedded(v); ok {
		if !b.allowed(v) {
			return nil, b.notPermitted(req.Path)
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
	if len(out) > maxListEntries {
		return withNotice(out[:maxListEntries], "listing truncated at %d entries; narrow the path", maxListEntries), nil
	}
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
	if len(out) > maxListEntries {
		return withNotice(out[:maxListEntries], "listing truncated at %d entries", maxListEntries), nil
	}
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
		return nil, b.notPermitted(req.Path)
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
	if len(out) > maxListEntries {
		return withNotice(out[:maxListEntries], "glob truncated at %d files; narrow the pattern or the path", maxListEntries), nil
	}
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
	if len(out) > maxListEntries {
		return withNotice(out[:maxListEntries], "glob truncated at %d files", maxListEntries), nil
	}
	return out, nil
}

// listFiles 枚举 base 下所有普通文件，供 GrepRaw 使用。不能复用 GlobInfo：它是
// 面向模型的工具、有 maxListEntries 封顶，grep 借道它会在大目录上静默丢掉排序靠后
// 的文件——"搜过了"变成漏报，不是省预算。遍历语义与 GlobInfo 保持一致：跳过软链
// 与隐藏目录，越出白名单的子树不计入。
func (b *Backend) listFiles(ctx context.Context, v string) ([]string, error) {
	if ep, ok := b.embedded(v); ok {
		var out []string
		err := fs.WalkDir(b.skillFS, ep, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			out = append(out, path.Join(v, strings.TrimPrefix(strings.TrimPrefix(p, ep), "/")))
			return nil
		})
		return out, err
	}
	full, err := b.resolve(ctx, v)
	if err != nil {
		return nil, err
	}
	var out []string
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
		child := path.Join(v, filepath.ToSlash(rel))
		if !b.allowed(child) {
			return nil
		}
		out = append(out, child)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func (b *Backend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	base := req.Path
	if base == "" {
		base = "/"
	}
	v, err := b.virtual(base)
	if err != nil {
		return nil, err
	}
	if !b.allowed(v) {
		return nil, b.notPermitted(req.Path)
	}
	files, err := b.listFiles(ctx, v)
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
	// 多行模式没实现。静默忽略会让跨行正则永远匹配不到、模型却以为搜过了，安全
	// 扫描里这就是漏报；显式报错让它改成单行模式或直接读文件。
	if req.EnableMultiline {
		return nil, errors.New("multiline grep is not supported: patterns match within a single line; read the file instead")
	}
	var globRe *regexp.Regexp
	if req.Glob != "" {
		globRe, err = GlobToRegexp(req.Glob)
		if err != nil {
			return nil, err
		}
	}
	fileType := strings.ToLower(strings.TrimPrefix(req.FileType, "."))
	before, after := req.BeforeLines, req.AfterLines
	var out []filesystem.GrepMatch
	for _, fp := range files {
		if globRe != nil && !globRe.MatchString(path.Base(fp)) && !globRe.MatchString(fp) {
			continue
		}
		if fileType != "" && strings.TrimPrefix(path.Ext(fp), ".") != fileType {
			continue
		}
		// 走 readAll 而不是 Read：Read 会给长文件封顶，用它搜索会漏掉 1000 行之后
		// 的全部命中。
		text, err := b.readAll(ctx, fp)
		if err != nil {
			continue
		}
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			out = append(out, filesystem.GrepMatch{
				Content: grepLine(line) + grepContext(lines, i, before, after),
				Path:    fp,
				Line:    i + 1,
			})
		}
	}
	// 命中数或总字节数封顶，避免一次 grep 把上下文预算吃光。
	//
	// 被削掉时补一条以 "[sandbox]" 为路径的说明项，而不是静默截断：模型必须知道
	// 自己看到的不是全部命中，否则"没搜到"会被当成"不存在"。用一条独立项而不是
	// 拼进最后一条命中的 Content，是因为上游默认的 files_with_matches / count 模式
	// 会把 Content 丢掉，只有独立项才在各模式下都可见。
	total := len(out)
	reason := ""
	if len(out) > maxGrepMatches {
		out, reason = out[:maxGrepMatches], fmt.Sprintf("match cap %d", maxGrepMatches)
	}
	if size := grepSize(out); size > maxGrepBytes {
		out = capGrepBytes(out, maxGrepBytes)
		reason = fmt.Sprintf("size cap %d KB", maxGrepBytes/1024)
	}
	if reason != "" {
		out = append(out, filesystem.GrepMatch{
			Path:    "[sandbox]",
			Content: fmt.Sprintf("grep truncated: showing %d of %d matches (%s); narrow the pattern or the path", len(out), total, reason),
		})
	}
	return out, nil
}

func grepSize(matches []filesystem.GrepMatch) int {
	size := 0
	for _, m := range matches {
		size += len(m.Path) + len(m.Content) + 8
	}
	return size
}

func capGrepBytes(matches []filesystem.GrepMatch, limit int) []filesystem.GrepMatch {
	size := grepSize(matches)
	for len(matches) > 1 && size > limit {
		last := matches[len(matches)-1]
		size -= len(last.Path) + len(last.Content) + 8
		matches = matches[:len(matches)-1]
	}
	return matches
}

// grepLine 给单行内容封顶。压缩过的 JS/Base64 常有一行几十 KB，整行回灌既没用
// 又占预算。
func grepLine(line string) string {
	if len(line) <= maxGrepLine {
		return line
	}
	return truncateBytes(line, maxGrepLine) + " …[line truncated]"
}

// grepContext 渲染命中行前后的上下文。GrepMatch 没有承载上下文行的字段，而上游的
// formatContentMatches 会把 Content 原样拼进 `path:line:content` 之后，所以在 Content
// 里插换行就能得到与 grep -A/-B 相近的可读结果。
func grepContext(lines []string, idx, before, after int) string {
	if before <= 0 && after <= 0 {
		return ""
	}
	var sb strings.Builder
	lo := idx - before
	if lo < 0 {
		lo = 0
	}
	hi := idx + after
	if hi > len(lines)-1 {
		hi = len(lines) - 1
	}
	for i := lo; i <= hi; i++ {
		if i == idx {
			continue
		}
		sb.WriteString("\n    ")
		sb.WriteString(grepLine(lines[i]))
	}
	return sb.String()
}

func (b *Backend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	return ErrReadOnly
}

func (b *Backend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	return ErrReadOnly
}

// GlobToRegexp 把 glob 转成 RE2 正则：** 跨目录、* 不跨、? 匹配单字符、[...] 字符类、
// {a,b} 花括号展开（可嵌套，对应 rg --glob 的语义）。
//
// 字符类与花括号是必需的，不是锦上添花：上游 glob/grep 的工具说明向模型承诺了
// `[abc]` 与 `'*.{ts,tsx}'`，不支持的话模型照写就是静默零结果，而它不会去怀疑模式
// 语法——只会以为目录里没有这类文件。
func GlobToRegexp(pattern string) (*regexp.Regexp, error) {
	alts := expandBraces(pattern)
	bodies := make([]string, 0, len(alts))
	for _, a := range alts {
		bodies = append(bodies, globBody(a))
	}
	// 顶层非捕获分组：花括号展开出的多个分支各自带 ^...$ 语义会互相打架，所以把
	// 断言提到分组外面。
	return regexp.Compile("^(?:" + strings.Join(bodies, "|") + ")$")
}

// expandBraces 展开 {a,b} 分支。没有花括号时原样返回，此时 '{' 会按字面量处理。
func expandBraces(pattern string) []string {
	depth, open := 0, -1
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
		case '{':
			if depth == 0 {
				open = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth > 0 {
				continue
			}
			var out []string
			for _, alt := range splitTopLevel(pattern[open+1 : i]) {
				for _, head := range expandBraces(alt) {
					for _, tail := range expandBraces(pattern[i+1:]) {
						out = append(out, pattern[:open]+head+tail)
					}
				}
			}
			return out
		}
	}
	return []string{pattern}
}

// splitTopLevel 按顶层逗号切分花括号内部，嵌套的 {} 与字符类里的逗号不算分隔符。
func splitTopLevel(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// globBody 把单个（已展开花括号的）glob 片段转成正则体，不带 ^ $ 断言。
func globBody(pattern string) string {
	var sb strings.Builder
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
		case '[':
			cls, end, ok := charClass(pattern, i)
			if !ok {
				// 未闭合的 '[' 是普通字符，不是语法错误。
				sb.WriteString(`\[`)
				continue
			}
			sb.WriteString(cls)
			i = end
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', ']', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// charClass 把 glob 的 [...] 翻译成正则字符类，返回类文本与 ']' 的下标。
// 否定类额外排除 '/'，因为 glob 的字符类跨不过目录分隔符。
func charClass(pattern string, start int) (string, int, bool) {
	i := start + 1
	negate := i < len(pattern) && (pattern[i] == '!' || pattern[i] == '^')
	if negate {
		i++
	}
	var body strings.Builder
	if i < len(pattern) && pattern[i] == ']' {
		// 首位的 ']' 属于字符类本身。
		body.WriteString(`\]`)
		i++
	}
	end := -1
	for ; i < len(pattern); i++ {
		if pattern[i] == ']' {
			end = i
			break
		}
		switch pattern[i] {
		case '\\', '^', '[', ']':
			body.WriteByte('\\')
		}
		body.WriteByte(pattern[i])
	}
	if end < 0 {
		return "", 0, false
	}
	var sb strings.Builder
	sb.WriteByte('[')
	if negate {
		sb.WriteString("^/")
	}
	sb.WriteString(body.String())
	sb.WriteByte(']')
	return sb.String(), end, true
}
