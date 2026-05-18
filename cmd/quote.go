package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
	"github.com/VecTrade-io/vectrade-cli/internal/output"
)

var quoteFields string

var quoteCmd = &cobra.Command{
	Use:   "quote [symbol]",
	Short: "Get a real-time stock quote",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuote,
}

func init() {
	quoteCmd.Flags().StringVarP(&quoteFields, "fields", "f", "", "comma-separated fields to return")
}

func runQuote(cmd *cobra.Command, args []string) error {
	symbol := args[0]

	cfg := config.Load(apiKey, sandbox, cfgFile)
	if outputFmt != "" {
		cfg.Output = outputFmt
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	params := make(map[string]string)
	if quoteFields != "" {
		params["fields"] = quoteFields
	}

	body, err := client.Get(context.Background(), "/vq/quotes/"+symbol, params)
	if err != nil {
		return err
	}

	outputFmt := output.ParseFormat(cfg.Output)
	if outputFmt == output.FormatJSON {
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}

	var quote struct {
		Symbol        string  `json:"symbol"`
		Price         float64 `json:"price"`
		Change        float64 `json:"change"`
		ChangePct     float64 `json:"change_pct"`
		Volume        int64   `json:"volume"`
		DayHigh       float64 `json:"day_high"`
		DayLow        float64 `json:"day_low"`
		PreviousClose float64 `json:"previous_close"`
	}
	if err := json.Unmarshal(body, &quote); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	tbl := output.NewTable("Field", "Value")
	tbl.AddRow("Symbol", quote.Symbol)
	tbl.AddRow("Price", fmt.Sprintf("$%.2f", quote.Price))
	tbl.AddRow("Change", fmt.Sprintf("%.2f (%.2f%%)", quote.Change, quote.ChangePct))
	tbl.AddRow("Volume", fmt.Sprintf("%d", quote.Volume))
	tbl.AddRow("Day Range", fmt.Sprintf("$%.2f – $%.2f", quote.DayLow, quote.DayHigh))
	tbl.AddRow("Prev Close", fmt.Sprintf("$%.2f", quote.PreviousClose))

	return tbl.Render(os.Stdout, outputFmt)
}
