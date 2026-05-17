//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/VecTrade-io/vectrade-cli/cmd"
)

func main() {
	if err := os.MkdirAll("completions", 0755); err != nil {
		log.Fatalf("creating completions directory: %v", err)
	}
	rootCmd := cmd.RootCmd()
	if err := rootCmd.GenBashCompletionFile("completions/vectrade.bash"); err != nil {
		log.Fatalf("generating bash completions: %v", err)
	}
	if err := rootCmd.GenZshCompletionFile("completions/vectrade.zsh"); err != nil {
		log.Fatalf("generating zsh completions: %v", err)
	}
	if err := rootCmd.GenFishCompletionFile("completions/vectrade.fish", true); err != nil {
		log.Fatalf("generating fish completions: %v", err)
	}
}
