package cmd

import (
	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
)

var (
	cfgFile   string
	apiKey    string
	sandbox   bool
	outputFmt string
)

// setAPIVersion propagates the build-time version to the API client package.
func setAPIVersion(v string) {
	api.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "vectrade",
	Short: "VecTrade CLI — financial data and AI from your terminal",
	Long: `VecTrade CLI provides access to real-time market data, AI-powered
analysis, and portfolio management directly from your terminal.

Get started:
  vectrade auth login
  vectrade quote AAPL
  vectrade ai "Analyze MSFT earnings"`,
}

func Execute() error {
	return rootCmd.Execute()
}

func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.vectrade/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides config/env)")
	rootCmd.PersistentFlags().BoolVar(&sandbox, "sandbox", false, "use sandbox environment")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "", "output format: table, json, csv")

	rootCmd.AddCommand(quoteCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(openapiCmd)
	rootCmd.AddCommand(usageCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(webhookCmd)
}
