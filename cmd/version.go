package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/auth"
)

// Build-time variables set via ldflags (see .goreleaser.yml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// Propagate build-time version to the API client and auth packages.
	setAPIVersion(version)
	auth.CLIVersion = version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vectrade %s\n", version)
		if commit != "none" {
			fmt.Printf("  commit: %s\n", commit)
		}
		if date != "unknown" {
			fmt.Printf("  built:  %s\n", date)
		}
		fmt.Printf("  go:     %s\n", runtime.Version())
		fmt.Printf("  os:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

// versionCmd is registered in root.go init() — no duplicate init() needed.
