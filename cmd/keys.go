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

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long:  `Create, list, and revoke API keys for your VecTrade account.`,
}

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE:  runKeysCreate,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE:  runKeysList,
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke [key-id]",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysRevoke,
}

var keyLabel string

func init() {
	keysCreateCmd.Flags().StringVar(&keyLabel, "label", "", "human-readable label for the key")
	keysCreateCmd.MarkFlagRequired("label")

	keysCmd.AddCommand(keysCreateCmd)
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysRevokeCmd)
	rootCmd.AddCommand(keysCmd)
}

type keyCreateResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

type keyEntry struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Prefix    string `json:"prefix"`
	CreatedAt string `json:"created_at"`
	LastUsed  string `json:"last_used"`
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	body, err := client.Post(context.Background(), "/vq/keys", map[string]string{"label": keyLabel})
	if err != nil {
		return err
	}

	var result keyCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Fprintf(os.Stdout, "API key created successfully!\n\n")
	fmt.Fprintf(os.Stdout, "  Key:     %s\n", result.Key)
	fmt.Fprintf(os.Stdout, "  ID:      %s\n", result.ID)
	fmt.Fprintf(os.Stdout, "  Label:   %s\n", result.Label)
	fmt.Fprintf(os.Stdout, "\n⚠️  Save this key — it won't be shown again.\n")
	return nil
}

func runKeysList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	body, err := client.Get(context.Background(), "/vq/keys", nil)
	if err != nil {
		return err
	}

	var result struct {
		Data []keyEntry `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLABEL\tPREFIX\tCREATED\tLAST USED")
	for _, k := range result.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Label, k.Prefix, k.CreatedAt, k.LastUsed)
	}
	return w.Flush()
}

func runKeysRevoke(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	err = client.Delete(context.Background(), fmt.Sprintf("/vq/keys/%s", args[0]))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Key %s revoked successfully.\n", args[0])
	return nil
}
