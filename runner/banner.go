package runner

import (
	"github.com/projectdiscovery/gologger"
)

// ASCII art 含反引号，无法放进 Go 的原始字符串字面量，因此用拼接的解释型字符串。
const banner = "\n" +
	"   _    __     __  _     \n" +
	"  | |  / /__  / /_(_)  __\n" +
	"  | | / / _ `/ __/ / `/_/\n" +
	"  | |/ /  __/ /_/ />  <  \n" +
	"  |___/\\___/\\__/_/_/|_|   %s\n"

func ShowBanner() {
	gologger.Print().Msgf(banner, Version)
}
