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
		// 参数校验失败即退出码 1，脚本据此判断扫描是否真的跑起来了。
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
