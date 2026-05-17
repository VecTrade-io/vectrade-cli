package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Manage OpenAPI specification",
	Long:  `Download, validate, and diff the VecTrade OpenAPI specification.`,
}

var openapiDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download the latest OpenAPI spec",
	Long:  `Download the current VecTrade OpenAPI 3.1 specification to a local file.`,
	RunE:  runOpenapiDownload,
}

var openapiDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes between local and remote spec",
	RunE:  runOpenapiDiff,
}

var (
	openapiOutput  string
	openapiVersion string
)

func init() {
	openapiDownloadCmd.Flags().StringVarP(&openapiOutput, "output", "o", "openapi.yaml", "output file path")
	openapiDownloadCmd.Flags().StringVar(&openapiVersion, "version", "latest", "spec version to download")

	openapiCmd.AddCommand(openapiDownloadCmd)
	openapiCmd.AddCommand(openapiDiffCmd)
	rootCmd.AddCommand(openapiCmd)
}

func runOpenapiDownload(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)

	specPath := "/vq/openapi"
	if openapiVersion != "latest" {
		specPath = fmt.Sprintf("/vq/openapi?version=%s", openapiVersion)
	}

	body, err := client.Get(context.Background(), specPath, nil)
	if err != nil {
		return fmt.Errorf("downloading spec: %w", err)
	}

	// Validate output path — prevent path traversal
	absOutput, err := filepath.Abs(openapiOutput)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	if !strings.HasPrefix(absOutput, cwd) {
		return fmt.Errorf("output path must be within current directory (got %s)", absOutput)
	}

	// Ensure output directory exists
	dir := filepath.Dir(absOutput)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	if err := os.WriteFile(absOutput, body, 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	info, _ := os.Stat(absOutput)
	fmt.Fprintf(os.Stdout, "✓ OpenAPI spec downloaded: %s (%d bytes)\n", openapiOutput, info.Size())
	return nil
}

func runOpenapiDiff(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Check if local file exists
	localPath := openapiOutput
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local spec not found at %s — run 'vectrade openapi download' first", localPath)
	}

	client := api.NewClient(cfg)
	remoteBody, err := client.Get(context.Background(), "/vq/openapi", nil)
	if err != nil {
		return fmt.Errorf("fetching remote spec: %w", err)
	}

	localBody, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("reading local spec: %w", err)
	}

	if string(localBody) == string(remoteBody) {
		fmt.Fprintln(os.Stdout, "✓ Local spec is up to date.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "⚠ Spec has changed (local: %d bytes, remote: %d bytes)\n", len(localBody), len(remoteBody))
	fmt.Fprintln(os.Stdout, "  Run 'vectrade openapi download' to update.")
	return nil
}
