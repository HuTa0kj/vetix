package runner

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmScanAnswers(t *testing.T) {
	for _, in := range []string{"y\n", "Y\n", "yes\n", "  YES  \n"} {
		var out bytes.Buffer
		ok, err := confirmScan(&out, strings.NewReader(in), "/skills", []string{"/skills/a"})
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", in, err)
		}
		if !ok {
			t.Errorf("input %q must be accepted", in)
		}
	}
	for _, in := range []string{"n\n", "no\n", "\n", "y yes\n", "\x1b[2Jevil\n"} {
		var out bytes.Buffer
		ok, err := confirmScan(&out, strings.NewReader(in), "/skills", []string{"/skills/a"})
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", in, err)
		}
		if ok {
			t.Errorf("input %q must be a refusal", in)
		}
	}
}

func TestConfirmScanEOFRefuses(t *testing.T) {
	// 非交互输入（管道已结束、定时任务）宁可取消也不误扫。
	var out bytes.Buffer
	ok, err := confirmScan(&out, strings.NewReader(""), "/skills", []string{"/skills/a"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("EOF must be treated as a refusal")
	}
}

func TestConfirmScanListing(t *testing.T) {
	var out bytes.Buffer
	ok, err := confirmScan(&out, strings.NewReader("n\n"), "/skills",
		[]string{"/skills/beta", "/skills/alpha"})
	if err != nil || ok {
		t.Fatalf("refusal path broken: ok=%v err=%v", ok, err)
	}
	outStr := out.String()
	if !strings.Contains(outStr, "Discovered Skills (2) under /skills") {
		t.Errorf("header with count and root missing:\n%s", outStr)
	}
	if !strings.Contains(outStr, "beta") || !strings.Contains(outStr, "alpha") {
		t.Errorf("skill names missing from the listing:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Scan all 2 skills? [y/N]") {
		t.Errorf("confirm prompt missing:\n%s", outStr)
	}
}

// 目录名可以是攻击者可控文本，清单渲染前必须过 sanitize。
func TestConfirmScanSanitizesName(t *testing.T) {
	var out bytes.Buffer
	if _, err := confirmScan(&out, strings.NewReader("n\n"), "/skills",
		[]string{"/skills/ev\x1b[2Jil"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b[2J") {
		t.Errorf("escape sequence must be stripped:\n%q", out.String())
	}
}
