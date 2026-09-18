package runner

import (
	"bytes"
	"io"
	"os"
)

// runinfoMarker 是 langsmith 回调在 OnStart 里打印的调试记录前缀。它输出整个 Run
// （含完整 prompt），单个 span 可达数十 KB，上游没有任何开关可关。
//
// 记录一定出现在行首（回调里是 fmt.Printf("...%+v\n", run)），过滤按行首判定。
const runinfoMarker = "[langsmith] runinfo:"

// installRuninfoFilter 让 stdout 在返回的 restore 被调用之前过滤掉 runinfo 记录，
// 其余字节原样透传。调用方必须调用 restore，它负责把缓冲区里的内容落盘，漏掉会
// 丢掉末尾的输出。
//
// 之所以能只在 stdout 上拦：vetix 自己的日志走 gologger → stderr，stdout 上只有
// 审计报告。报告内容本身可能包含这个前缀（例如扫到 vetix 自己的源码），所以过滤
// 必须按行首判定，见 filterChunk。
func installRuninfoFilter() func() {
	real := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return func() {}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer r.Close()
		filterRuninfo(r, real)
	}()

	os.Stdout = w
	return func() {
		os.Stdout = real
		w.Close()
		<-done
	}
}

// filterRuninfo 把 src 复制到 dst，只丢掉 runinfo 记录本身。
func filterRuninfo(src io.Reader, dst io.Writer) {
	var (
		pending []byte
		tmp     = make([]byte, 32*1024)
	)
	for {
		n, err := src.Read(tmp)
		if n > 0 {
			pending = append(pending, tmp[:n]...)
			pending = filterChunk(pending, dst)
		}
		if err != nil {
			break
		}
	}
	// 尾部没有换行的内容：以记录前缀开头说明是一条被截断的记录（读中断），丢掉；
	// 否则是报告的收尾，原样落盘。
	if len(pending) > 0 && !bytes.HasPrefix(pending, []byte(runinfoMarker)) {
		_, _ = dst.Write(pending)
	}
}

// filterChunk 逐行判定并输出，返回最后一段没有换行、还不能确定是否完整的尾巴。
//
// 只丢弃"整行以记录前缀开头"的行，不做子串匹配：报告文本里也可能出现这个前缀
// （自扫 vetix 仓库就会命中 runner/quiet.go 里那一行），子串匹配会把该行从前缀处
// 截断、还会把两个换行并成一个，表格当场错位。记录本身永远是整行输出（回调里是
// fmt.Printf("...%+v\n")），所以行首判定足够，也不会误杀报告内容。
func filterChunk(buf []byte, dst io.Writer) []byte {
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			return buf
		}
		line := buf[:i+1]
		if !bytes.HasPrefix(line, []byte(runinfoMarker)) {
			_, _ = dst.Write(line)
		}
		buf = buf[i+1:]
	}
}
