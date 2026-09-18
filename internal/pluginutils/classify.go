package pluginutils

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// AllowedExt 来自 OpenClaw 的上传白名单，与 Python 版逐项一致。
var AllowedExt = map[string]bool{
	".md": true, ".mdx": true, ".txt": true, ".json": true, ".json5": true,
	".yaml": true, ".yml": true, ".toml": true,
	".js": true, ".cjs": true, ".mjs": true, ".ts": true, ".tsx": true, ".jsx": true,
	".py": true, ".sh": true, ".rb": true, ".go": true,
	".rs": true, ".swift": true, ".kt": true, ".java": true, ".cs": true,
	".cpp": true, ".c": true, ".h": true, ".hpp": true,
	".sql": true, ".csv": true, ".ini": true, ".cfg": true, ".xml": true,
	".html": true, ".css": true, ".scss": true, ".sass": true,
}

// Ext 复刻 Python 的 os.path.splitext 语义：以点开头的文件名本身不算扩展名，
// 所以 ".env" 的扩展名是空串（会被判为 rare file），而 ".env.txt" 是 ".txt"。
// Go 的 path.Ext 会把 ".env" 返回 ".env"，语义不同，不能用。
func Ext(filePath string) string {
	base := filepath.Base(filePath)
	i := strings.LastIndex(base, ".")
	if i <= 0 {
		return ""
	}
	return strings.ToLower(base[i:])
}

func IsRiskFile(filePath string) bool {
	return !AllowedExt[Ext(filePath)]
}

// textChars 复刻 Python 版的字节集合：
// {7,8,9,10,12,13,27} | (range(0x20,0x100) - {0x7f})
var textChars [256]bool

func init() {
	for _, b := range []byte{7, 8, 9, 10, 12, 13, 27} {
		textChars[b] = true
	}
	for b := 0x20; b <= 0xff; b++ {
		textChars[byte(b)] = true
	}
	textChars[0x7f] = false
}

func ExistNonText(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	nonText := 0
	for _, b := range content {
		if !textChars[b] {
			nonText++
		}
	}
	return float64(nonText)/float64(len(content)) > 0.3
}

// IsBinaryFile 只读前 1KB 判定：含 NUL，或非文本字节占比超阈值。
func IsBinaryFile(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()
	chunk := make([]byte, 1024)
	n, _ := f.Read(chunk)
	chunk = chunk[:n]
	for _, b := range chunk {
		if b == 0 {
			return true
		}
	}
	return ExistNonText(chunk)
}

// DecodeReplace 复刻 Python 的 read_text(encoding="utf-8", errors="replace")：
// 非法字节被替换成 U+FFFD，因此字节长度可能与磁盘上的原始长度不同。
// 插件里凡是用到"文件大小/行数"的地方都必须走这个函数，否则与 Python 不一致。
func DecodeReplace(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	var sb strings.Builder
	sb.Grow(len(raw))
	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRune(raw[i:])
		if r == utf8.RuneError && size == 1 {
			sb.WriteRune(utf8.RuneError)
			i++
			continue
		}
		sb.Write(raw[i : i+size])
		i += size
	}
	return sb.String()
}

// SplitLines 对齐 Python str.splitlines()：末尾换行不产生额外空行，
// \r\n / \r / \v / \f / \x1c-\x1e / \x85 / \u2028 / \u2029 都算换行。
func SplitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if isLineBreak(r) {
			lines = append(lines, s[start:i])
			i += size
			if r == '\r' && i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// HumanBytes 对齐 Python 插件的 human_display 输出格式。
func HumanBytes(n int) string {
	switch {
	case n < 1024:
		return strconv.Itoa(n) + " B"
	case n < 1024*1024:
		return strconv.FormatFloat(float64(n)/1024, 'f', 1, 64) + " KB"
	default:
		return strconv.FormatFloat(float64(n)/(1024*1024), 'f', 1, 64) + " MB"
	}
}

// CleanVirtual 把路径统一成以 / 开头的斜杠形式，用于与 Python 的
// os.path.relpath 输出对齐。
func CleanVirtual(p string) string {
	return path.Clean(filepath.ToSlash(p))
}
