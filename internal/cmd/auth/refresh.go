package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// RefreshOptions holds the options for the refresh command.
type RefreshOptions struct {
	IO       *iostreams.IOStreams
	Hostname string
}

// NewCmdRefresh creates the refresh command.
func NewCmdRefresh(ios *iostreams.IOStreams) *cobra.Command {
	opts := &RefreshOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh authentication tokens",
		Long: `Force a refresh of the authentication tokens for an Atlassian host.

This command exchanges the stored refresh token for a new access token.
Use this when your access token has expired or is about to expire.

The refresh token is obtained during initial login (via 'atl auth login')
when the 'offline_access' scope is requested.`,
		Example: `  # Refresh tokens for current host
  atl auth refresh

  # Refresh tokens for a specific host
  atl auth refresh --hostname mycompany.atlassian.net`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefresh(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The hostname to refresh tokens for (defaults to current host)")

	return cmd
}

func runRefresh(opts *RefreshOptions) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine hostname
	hostname := opts.Hostname
	if hostname == "" {
		hostname = cfg.CurrentHost
	}
	if hostname == "" {
		return fmt.Errorf("no host specified and no current host configured\n\nRun 'atl auth login' first or specify --hostname")
	}

	// Check if host exists in config
	hostConfig := cfg.GetHost(hostname)
	if hostConfig == nil {
		return fmt.Errorf("no configuration found for host %s\n\nRun 'atl auth login --hostname %s' first", hostname, hostname)
	}

	// Get OAuth credentials
	clientID := os.Getenv("ATLASSIAN_CLIENT_ID")
	clientSecret := os.Getenv("ATLASSIAN_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		if cfg.OAuth != nil {
			if clientID == "" {
				clientID = cfg.OAuth.ClientID
			}
			if clientSecret == "" {
				clientSecret = cfg.OAuth.ClientSecret
			}
		}
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("OAuth credentials not configured\n\nRun 'atl auth setup' to configure your OAuth app credentials")
	}

	// Check current token status
	currentTokens, err := auth.GetToken(hostname)
	if err != nil {
		return fmt.Errorf("failed to get current tokens: %w", err)
	}
	if currentTokens == nil {
		return fmt.Errorf("no tokens found for %s\n\nRun 'atl auth login' first", hostname)
	}

	fmt.Fprintf(opts.IO.Out, "Refreshing tokens for %s...\n", hostname)

	// Show current token status
	if currentTokens.IsExpired() {
		fmt.Fprintln(opts.IO.Out, "Current token: expired")
	} else {
		remaining := time.Until(currentTokens.ExpiresAt)
		fmt.Fprintf(opts.IO.Out, "Current token: expires in %s\n", formatDuration(remaining))
	}

	// Refresh tokens
	ctx := context.Background()
	newTokens, err := auth.RefreshAccessToken(ctx, hostname, &auth.RefreshConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to refresh tokens: %w", err)
	}

	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, output.Success.Render("Tokens refreshed successfully!"))
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintf(opts.IO.Out, "New token expires: %s\n", newTokens.ExpiresAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(opts.IO.Out, "Valid for: %s\n", formatDuration(time.Until(newTokens.ExpiresAt)))

	return nil
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
