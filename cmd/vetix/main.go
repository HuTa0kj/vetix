package main

import (
	"os"

	"github.com/projectdiscovery/gologger"

	"vetix/runner"
)

func main() {
	options, err := runner.ParseOptions()
	if err != nil {
		gologger.Error().Msg(err.Error())
		// GoTStarter 的错误路径恒返回 0；vetix 的既有行为是参数校验失败即退出码 1，
		// 这里保留后者，否则脚本无法判断扫描是否真的跑起来了。
		os.Exit(1)
	}
	taskRunner, err := runner.New(options)
	if err != nil {
		gologger.Error().Msg(err.Error())
		os.Exit(1)
	}
	if err := taskRunner.Run(); err != nil {
		gologger.Error().Msg(err.Error())
		os.Exit(1)
	}
}
