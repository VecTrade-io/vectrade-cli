//go:build ignore

package main

import (
	"os"

	"github.com/VecTrade-io/vectrade-cli/cmd"
)

func main() {
	os.MkdirAll("completions", 0755)
	rootCmd := cmd.RootCmd()
	rootCmd.GenBashCompletionFile("completions/vectrade.bash")
	rootCmd.GenZshCompletionFile("completions/vectrade.zsh")
	rootCmd.GenFishCompletionFile("completions/vectrade.fish", true)
}
