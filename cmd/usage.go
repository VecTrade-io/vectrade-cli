package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "View API usage and quota",
	Long:  `Display current billing period API usage, limits, and remaining quota.`,
	RunE:  runUsage,
}

var usageThisMonth bool

func init() {
	usageCmd.Flags().BoolVar(&usageThisMonth, "this-month", true, "show current billing month")
}

type usageResponse struct {
	Period         string `json:"period"`
	PlanName       string `json:"plan_name"`
	TotalRequests  int    `json:"total_requests"`
	QuotaLimit     int    `json:"quota_limit"`
	TokensUsed     int    `json:"tokens_used"`
	TokenQuota     int    `json:"token_quota"`
	TokenRemaining int    `json:"token_remaining"`
	OverageCalls   int    `json:"overage_calls"`
}

func runUsage(cmd *cobra.Command, args []string) error {
	cfg := config.Load(apiKey, sandbox, cfgFile)
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	body, err := client.Get(context.Background(), "/vq/developer/usage", nil)
	if err != nil {
		return err
	}

	var result usageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Plan: %s | Period: %s\n\n", result.PlanName, result.Period)

	var requestsPct float64
	if result.QuotaLimit > 0 {
		requestsPct = float64(result.TotalRequests) / float64(result.QuotaLimit) * 100
	}

	fmt.Fprintf(os.Stdout, "Requests:  %d / %d (%.1f%%)\n", result.TotalRequests, result.QuotaLimit, requestsPct)
	fmt.Fprintf(os.Stdout, "Tokens:    %d / %d (%d remaining)\n", result.TokensUsed, result.TokenQuota, result.TokenRemaining)
	if result.OverageCalls > 0 {
		fmt.Fprintf(os.Stdout, "Overage calls: %d\n", result.OverageCalls)
	}

	return nil
}
