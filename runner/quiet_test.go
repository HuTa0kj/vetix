package runner

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// filter 跑一次过滤，返回结果。
func filter(t *testing.T, in string, chunk int) string {
	t.Helper()
	var out bytes.Buffer
	src := &chunkedReader{s: in, n: chunk}
	filterRuninfo(src, &out)
	return out.String()
}

// chunkedReader 按固定大小投喂，用来模拟记录被读断在中间。
type chunkedReader struct {
	s string
	n int
	i int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, os.ErrClosed
	}
	end := r.i + r.n
	if end > len(r.s) {
		end = len(r.s)
	}
	n := copy(p, r.s[r.i:end])
	r.i += n
	return n, nil
}

func TestFilterDropsRuninfoRecord(t *testing.T) {
	in := "REPORT LINE\n" +
		"[langsmith] runinfo: &{ID:abc Name:Lambda Inputs:map[input:{}]}\n" +
		"MORE REPORT\n"
	want := "REPORT LINE\nMORE REPORT\n"
	for _, chunk := range []int{1, 2, 7, 13, 64, 4096} {
		if got := filter(t, in, chunk); got != want {
			t.Errorf("chunk=%d:\n got %q\nwant %q", chunk, got, want)
		}
	}
}

func TestFilterDropsSeveralRecords(t *testing.T) {
	in := "[langsmith] runinfo: one\nkeep-1\n[langsmith] runinfo: two\nkeep-2\n"
	want := "keep-1\nkeep-2\n"
	for _, chunk := range []int{1, 3, 5, 11, 4096} {
		if got := filter(t, in, chunk); got != want {
			t.Errorf("chunk=%d: got %q want %q", chunk, got, want)
		}
	}
}

// 记录在多字节边界处被读断时，前缀不能漏出去。
func TestFilterHandlesSplitMarker(t *testing.T) {
	in := "[langsmith] runinfo: payload\nB"
	want := "B"
	for chunk := 1; chunk <= len(in); chunk++ {
		if got := filter(t, in, chunk); got != want {
			t.Errorf("chunk=%d: got %q want %q", chunk, got, want)
		}
	}
}

// 记录没带结尾换行（被截断的输出）时，剩下部分一并丢弃，不能漏到终端。
func TestFilterDropsUnterminatedRecord(t *testing.T) {
	if got := filter(t, "ok\n[langsmith] runinfo: partial payload", 4096); got != "ok\n" {
		t.Errorf("got %q", got)
	}
}

// 只有在行首的 marker 才是记录。报告行里出现同样的文本时，整行必须逐字节保留——
// 子串匹配会把该行从前缀处截断并把两个换行并成一个，表格当场错位。
func TestFilterKeepsMarkerTextMidLine(t *testing.T) {
	in := "│ 12 │ high │ dumps [langsmith] runinfo: to stdout │ SKILL.md │\nNEXT ROW\n"
	for _, chunk := range []int{1, 4, 9, 64, 4096} {
		if got := filter(t, in, chunk); got != in {
			t.Errorf("chunk=%d:\n got %q\nwant %q", chunk, got, in)
		}
	}
}

// 报告与记录交错写入时，报告必须逐字节保留。
func TestFilterPreservesInterleavedReport(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("│ report row ")
		b.WriteString(strings.Repeat("x", i%17))
		b.WriteString(" │\n")
		if i%3 == 0 {
			b.WriteString("[langsmith] runinfo: &{ID:big Name:Lambda Inputs:map[input:")
			b.WriteString(strings.Repeat("y", 500))
			b.WriteString("]}\n")
		}
	}
	// 期望值：按行剔除记录后重新拼接，保留原换行结构。
	var kept []string
	for _, line := range strings.Split(b.String(), "\n") {
		if strings.HasPrefix(line, runinfoMarker) {
			continue
		}
		kept = append(kept, line)
	}
	want := strings.Join(kept, "\n")
	got := filter(t, b.String(), 512)
	if got != want {
		t.Fatalf("interleaved output mismatch: got %d bytes want %d bytes", len(got), len(want))
	}
}

func TestFilterPassesCleanOutput(t *testing.T) {
	in := "no records here\njust the report\n"
	for _, chunk := range []int{1, 5, 4096} {
		if got := filter(t, in, chunk); got != in {
			t.Errorf("chunk=%d: clean output must pass through unchanged, got %q", chunk, got)
		}
	}
}
