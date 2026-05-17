package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

var aiCmd = &cobra.Command{
	Use:   "ai [prompt]",
	Short: "Run AI-powered financial analysis",
	Args:  cobra.ExactArgs(1),
	RunE:  runAI,
}

func runAI(cmd *cobra.Command, args []string) error {
	prompt := args[0]

	cfg := config.Load(apiKey, sandbox, cfgFile)
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg)

	body, err := client.StreamPost(context.Background(), "/vq/ai/analyze", map[string]any{
		"prompt": prompt,
		"stream": true,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			fmt.Fprintln(os.Stdout)
			break
		}

		var chunk struct {
			Content string `json:"content"`
			Type    string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Content != "" {
			fmt.Fprint(os.Stdout, chunk.Content)
		}
	}

	return scanner.Err()
}
