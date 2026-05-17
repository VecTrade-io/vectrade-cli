package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhooks",
	Long:  `Create, list, delete webhooks and listen for events locally.`,
}

var webhookListenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for webhook events locally (like stripe listen)",
	Long: `Start a local webhook listener that forwards events from VecTrade
to your local development server. Useful for testing webhook integrations.`,
	RunE: runWebhookListen,
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered webhooks",
	RunE:  runWebhookList,
}

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook subscription",
	RunE:  runWebhookCreate,
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete [webhook-id]",
	Short: "Delete a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookDelete,
}

var (
	listenPort    int
	listenForward string
	webhookURL    string
	webhookEvents []string
)

func init() {
	webhookListenCmd.Flags().IntVar(&listenPort, "port", 4242, "local port to listen on")
	webhookListenCmd.Flags().StringVar(&listenForward, "forward-to", "", "forward events to this URL")

	webhookCreateCmd.Flags().StringVar(&webhookURL, "url", "", "webhook delivery URL")
	webhookCreateCmd.Flags().StringSliceVar(&webhookEvents, "events", nil, "events to subscribe to")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.MarkFlagRequired("events")

	webhookCmd.AddCommand(webhookListenCmd)
	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookCreateCmd)
	webhookCmd.AddCommand(webhookDeleteCmd)
	rootCmd.AddCommand(webhookCmd)
}

type webhookEntry struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

func runWebhookListen(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "⚡ Listening for webhook events on port %d...\n", listenPort)
	if listenForward != "" {
		fmt.Fprintf(os.Stdout, "   Forwarding to: %s\n", listenForward)
	}
	fmt.Fprintln(os.Stdout, "   Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := streamWebhookEvents(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", err)
		}
	}()

	<-sigChan
	fmt.Fprintln(os.Stdout, "\n✋ Stopped listening.")
	return nil
}

func streamWebhookEvents(cfg *config.Config) error {
	client := api.NewClient(cfg)
	body, err := client.StreamGet(context.Background(), "/vq/webhooks/listen")
	if err != nil {
		return err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				eventType, _ := event["type"].(string)
				timestamp := time.Now().Format("15:04:05")
				fmt.Fprintf(os.Stdout, "[%s] %s → %s\n", timestamp, eventType, data)

				if listenForward != "" {
					go forwardEvent(listenForward, data)
				}
			}
		}
	}
	return scanner.Err()
}

func forwardEvent(targetURL, data string) {
	// Only allow forwarding to localhost URLs to prevent SSRF
	if !strings.HasPrefix(targetURL, "http://localhost") &&
		!strings.HasPrefix(targetURL, "http://127.0.0.1") &&
		!strings.HasPrefix(targetURL, "http://[::1]") {
		fmt.Fprintf(os.Stderr, "  ⚠ Forward blocked: only localhost URLs are allowed\n")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Forward failed: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Forward failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Fprintf(os.Stdout, "  → Forwarded (status %d)\n", resp.StatusCode)
}

func runWebhookList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	body, err := client.Get(context.Background(), "/vq/webhooks", nil)
	if err != nil {
		return err
	}

	var result struct {
		Data []webhookEntry `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No webhooks registered.")
		return nil
	}

	for _, wh := range result.Data {
		status := "active"
		if !wh.Active {
			status = "inactive"
		}
		fmt.Fprintf(os.Stdout, "%s  [%s]  %s\n", wh.ID, status, wh.URL)
		fmt.Fprintf(os.Stdout, "  Events: %s\n\n", strings.Join(wh.Events, ", "))
	}
	return nil
}

func runWebhookCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	payload := map[string]interface{}{
		"url":    webhookURL,
		"events": webhookEvents,
	}
	body, err := client.Post(context.Background(), "/vq/webhooks", payload)
	if err != nil {
		return err
	}

	var result webhookEntry
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Webhook created: %s\n", result.ID)
	fmt.Fprintf(os.Stdout, "  URL:    %s\n", result.URL)
	fmt.Fprintf(os.Stdout, "  Events: %s\n", strings.Join(result.Events, ", "))
	return nil
}

func runWebhookDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(apiKey, sandbox, cfgFile)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)
	err = client.Delete(context.Background(), fmt.Sprintf("/vq/webhooks/%s", args[0]))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Webhook %s deleted.\n", args[0])
	return nil
}
