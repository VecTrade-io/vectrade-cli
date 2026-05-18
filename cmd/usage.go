package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
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
	Period        string          `json:"period"`
	PlanName      string          `json:"plan_name"`
	RequestsUsed  int            `json:"requests_used"`
	RequestsLimit int            `json:"requests_limit"`
	CreditsUsed   float64        `json:"credits_used"`
	CreditsLimit  float64        `json:"credits_limit"`
	Endpoints     []endpointUsage `json:"endpoints"`
}

type endpointUsage struct {
	Name     string  `json:"name"`
	Requests int     `json:"requests"`
	Pct      float64 `json:"pct"`
}

func runUsage(cmd *cobra.Command, args []string) error {
	cfg := config.Load(apiKey, sandbox, cfgFile)
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	body, err := client.Get(context.Background(), "/vq/usage", nil)
	if err != nil {
		return err
	}

	var result usageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Plan: %s | Period: %s\n\n", result.PlanName, result.Period)

	var requestsPct, creditsPct float64
	if result.RequestsLimit > 0 {
		requestsPct = float64(result.RequestsUsed) / float64(result.RequestsLimit) * 100
	}
	if result.CreditsLimit > 0 {
		creditsPct = result.CreditsUsed / result.CreditsLimit * 100
	}

	fmt.Fprintf(os.Stdout, "Requests:  %d / %d (%.1f%%)\n", result.RequestsUsed, result.RequestsLimit, requestsPct)
	fmt.Fprintf(os.Stdout, "Credits:   %.2f / %.2f (%.1f%%)\n\n", result.CreditsUsed, result.CreditsLimit, creditsPct)

	if len(result.Endpoints) > 0 {
		fmt.Fprintln(os.Stdout, "Top endpoints:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ENDPOINT\tREQUESTS\tSHARE")
		for _, ep := range result.Endpoints {
			fmt.Fprintf(w, "%s\t%d\t%.1f%%\n", ep.Name, ep.Requests, ep.Pct)
		}
		w.Flush()
	}

	return nil
}
