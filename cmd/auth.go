package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/VecTrade-io/vectrade-cli/internal/auth"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

var (
	authProvider string
	errFmtCreds  = "reading credentials: %w"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long: `Authenticate with VecTrade using browser-based OAuth.

After authenticating, the CLI stores JWT credentials locally and uses them
for developer operations (key management, usage queries). For API data
access, use API keys via 'vectrade keys create'.

Examples:
  vectrade auth login                     # Login with Google (default)
  vectrade auth login --provider microsoft
  vectrade auth status                    # Show current auth status
  vectrade auth token                     # Print current access token
  vectrade auth logout                    # Clear stored credentials`,
}

// ── auth login ──────────────────────────────────────────────────────────

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with VecTrade via browser-based OAuth",
	Long: `Opens your browser for OAuth authentication (similar to 'gcloud auth login'
or 'gh auth login --web').

The CLI starts a local HTTP server, opens your browser to the OAuth provider,
and waits for the redirect with the authorization code. Tokens are stored
securely in your config directory with owner-only permissions.

Supported providers: google, microsoft, apple, x`,
	RunE: runAuthLogin,
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	provider := strings.ToLower(authProvider)

	// Validate provider
	valid := false
	for _, p := range auth.SupportedProviders {
		if p == provider {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unsupported provider %q — supported: %s", provider, strings.Join(auth.SupportedProviders, ", "))
	}

	// Resolve base URL
	cfg, err := config.Load("", sandbox, cfgFile)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  VecTrade CLI — Browser Authentication")
	fmt.Println("  ─────────────────────────────────────")
	fmt.Printf("  Provider:    %s\n", provider)
	fmt.Printf("  API:         %s\n", cfg.BaseURL)
	fmt.Println()

	creds, err := auth.Login(auth.LoginOptions{
		Provider: provider,
		BaseURL:  cfg.BaseURL,
		Timeout:  5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Authentication successful!")
	fmt.Printf("  ▸ Credentials stored at: %s\n", auth.CredentialsFilePath())
	fmt.Printf("  ▸ Session ID: %s\n", creds.SessionID)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    vectrade auth status    — verify authentication")
	fmt.Println("    vectrade keys create    — create an API key")
	fmt.Println("    vectrade usage          — check your quota")
	fmt.Println()

	return nil
}

// ── auth logout ─────────────────────────────────────────────────────────

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored authentication credentials",
	RunE:  runAuthLogout,
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return fmt.Errorf(errFmtCreds, err)
	}
	if creds == nil {
		fmt.Println("Not currently authenticated.")
		return nil
	}

	if err := auth.ClearCredentials(); err != nil {
		return fmt.Errorf("clearing credentials: %w", err)
	}

	fmt.Printf("Logged out. Credentials removed from %s\n", auth.CredentialsFilePath())
	return nil
}

// ── auth status ─────────────────────────────────────────────────────────

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return fmt.Errorf(errFmtCreds, err)
	}
	if creds == nil {
		fmt.Println("Not authenticated. Run 'vectrade auth login' to authenticate.")
		return nil
	}

	status := "valid"
	if creds.IsExpired() {
		status = "expired (will auto-refresh)"
	}

	fmt.Println()
	fmt.Println("  VecTrade CLI — Authentication Status")
	fmt.Println("  ────────────────────────────────────")
	fmt.Printf("  Status:      %s\n", status)
	fmt.Printf("  Provider:    %s\n", creds.Provider)
	fmt.Printf("  Session:     %s\n", creds.SessionID)
	fmt.Printf("  API:         %s\n", creds.BaseURL)
	fmt.Printf("  Credentials: %s\n", auth.CredentialsFilePath())
	if !creds.ExpiresAt.IsZero() {
		fmt.Printf("  Expires:     %s\n", creds.ExpiresAt.Format(time.RFC3339))
	}
	fmt.Println()

	return nil
}

// ── auth token ──────────────────────────────────────────────────────────

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current access token (for piping to other tools)",
	Long: `Prints the current access token to stdout. Useful for piping to other
tools or scripts:

  curl -H "Authorization: Bearer $(vectrade auth token)" https://api.vectrade.io/v1/...

If the token is expired, it is automatically refreshed.`,
	RunE: runAuthToken,
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return fmt.Errorf(errFmtCreds, err)
	}
	if creds == nil {
		return fmt.Errorf("not authenticated — run 'vectrade auth login'")
	}

	// Auto-refresh if expired
	if creds.IsExpired() {
		creds, err = auth.RefreshAccessToken(creds.BaseURL, creds)
		if err != nil {
			return err
		}
	}

	// Print only the token (no newline noise) so it can be captured by $()
	fmt.Fprint(os.Stdout, creds.AccessToken)
	return nil
}

// ── init ────────────────────────────────────────────────────────────────

func init() {
	authLoginCmd.Flags().StringVar(&authProvider, "provider", "google", "OAuth provider (google, microsoft, apple, x)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authTokenCmd)
}
