package runner

import (
	"bytes"
	"io"
	"os"
)

// runinfoMarker 是 langsmith 回调在 OnStart 里打印的调试记录前缀。它输出整个 Run
// （含完整 prompt），单个 span 可达数十 KB，上游没有任何开关可关。
const runinfoMarker = "[langsmith] runinfo:"

// installRuninfoFilter 让 stdout 在返回的 restore 被调用之前过滤掉 runinfo 记录，
// 其余字节原样透传。调用方必须调用 restore，它负责把缓冲区里的内容落盘，漏掉会
// 丢掉末尾的输出。
//
// 之所以能只在 stdout 上拦：vetix 自己的日志走 gologger → stderr，stdout 上只有
// 审计报告，而报告不含这个前缀，会原样穿过滤镜。
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
		pending  []byte
		tmp      = make([]byte, 32*1024)
		skipping bool
	)
	for {
		n, err := src.Read(tmp)
		if n > 0 {
			pending = append(pending, tmp[:n]...)
			pending = filterChunk(pending, dst, &skipping)
		}
		if err != nil {
			break
		}
	}
	// 记录若被截断则一直处于 skipping 且缓冲区为空，这里自然什么都不落盘。
	if len(pending) > 0 {
		_, _ = dst.Write(pending)
	}
}

// filterChunk 处理缓冲区里已经能确定的部分，返回尚未确定、需要留到下次的尾巴。
//
// 丢弃的是记录所占的字节区间而不是整行：报告也写 stdout，可能与记录交错，按行
// 过滤会连报告一起丢掉。被丢弃的记录自带的那个换行一并删掉，不额外补，因此被
// 记录切开的两次写入能拼回原样。
func filterChunk(buf []byte, dst io.Writer, skipping *bool) []byte {
	for {
		if *skipping {
			i := bytes.IndexByte(buf, '\n')
			if i < 0 {
				// 记录尚未结束，整段都还不能确定。
				return buf[:0]
			}
			buf = buf[i+1:]
			*skipping = false
			continue
		}

		i := bytes.Index(buf, []byte(runinfoMarker))
		if i < 0 {
			// 末尾可能是被读断的记录前缀，多留 len(marker)-1 个字节再判。
			keep := len(buf) - len(runinfoMarker) + 1
			if keep <= 0 {
				return buf
			}
			_, _ = dst.Write(buf[:keep])
			return append(buf[:0], buf[keep:]...)
		}
		if i > 0 {
			_, _ = dst.Write(buf[:i])
		}
		buf = buf[i:]
		*skipping = true
	}
}
