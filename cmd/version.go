package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vectrade %s\n", api.Version)
		fmt.Printf("  go:     %s\n", runtime.Version())
		fmt.Printf("  os:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
