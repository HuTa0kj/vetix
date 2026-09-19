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
