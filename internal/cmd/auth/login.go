package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// LoginOptions holds the options for the login command.
type LoginOptions struct {
	IO       *iostreams.IOStreams
	Hostname string
	Scopes   []string
}

// NewCmdLogin creates the login command.
func NewCmdLogin(ios *iostreams.IOStreams) *cobra.Command {
	opts := &LoginOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with an Atlassian host",
		Long: `Authenticate with an Atlassian Cloud instance.

This will open a browser window where you can authorize the CLI to access
your Atlassian account. The authorization tokens are stored securely in
your system's keychain/credential manager.`,
		Example: `  # Login to your Atlassian instance
  atl auth login

  # Login to a specific instance
  atl auth login --hostname mycompany.atlassian.net`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The hostname of the Atlassian instance to authenticate with")
	cmd.Flags().StringSliceVar(&opts.Scopes, "scopes", nil, "Additional OAuth scopes to request")

	return cmd
}

func runLogin(opts *LoginOptions) error {
	// Get OAuth credentials: env vars take precedence, then config file
	clientID := os.Getenv("ATLASSIAN_CLIENT_ID")
	clientSecret := os.Getenv("ATLASSIAN_CLIENT_SECRET")

	// If not in env, try config file
	if clientID == "" || clientSecret == "" {
		cfg, err := config.Load()
		if err == nil && cfg.OAuth != nil {
			if clientID == "" {
				clientID = cfg.OAuth.ClientID
			}
			if clientSecret == "" {
				clientSecret = cfg.OAuth.ClientSecret
			}
		}
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("OAuth credentials not configured.\n\nRun 'atl auth setup' to configure your OAuth app credentials.\n\nAlternatively, set ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET environment variables.")
	}

	scopes := auth.DefaultScopes()
	if len(opts.Scopes) > 0 {
		scopes = append(scopes, opts.Scopes...)
	}

	oauthConfig := &auth.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/callback", auth.DefaultCallbackPort),
		Scopes:       scopes,
	}

	flow, err := auth.NewOAuthFlow(oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize OAuth flow: %w", err)
	}

	// Start callback server on fixed port
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server, _, err := auth.StartCallbackServer(codeChan, errChan, flow.State())
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Open browser
	authURL := flow.AuthorizationURL()
	fmt.Fprintln(opts.IO.Out, "Opening browser to authenticate...")
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, "If the browser doesn't open, visit this URL:")
	fmt.Fprintln(opts.IO.Out, authURL)
	fmt.Fprintln(opts.IO.Out, "")

	if err := auth.OpenBrowser(authURL); err != nil {
		fmt.Fprintln(opts.IO.ErrOut, "Warning: Could not open browser automatically")
	}

	fmt.Fprintln(opts.IO.Out, "Waiting for authentication...")

	// Wait for callback
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		return fmt.Errorf("authentication failed: %w", err)
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timed out")
	}

	// Exchange code for tokens
	ctx := context.Background()
	tokens, err := flow.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Get accessible resources to find cloud ID
	resources, err := api.GetAccessibleResources(ctx, tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get accessible resources: %w", err)
	}

	if len(resources) == 0 {
		return fmt.Errorf("no accessible Atlassian sites found. Make sure your OAuth app has the correct permissions")
	}

	// Select resource (use first one or match hostname)
	var selectedResource *api.AccessibleResource
	for _, r := range resources {
		hostname := strings.TrimPrefix(r.URL, "https://")
		if opts.Hostname == "" || hostname == opts.Hostname {
			selectedResource = r
			break
		}
	}

	if selectedResource == nil {
		if opts.Hostname != "" {
			return fmt.Errorf("site %s not found in accessible resources", opts.Hostname)
		}
		selectedResource = resources[0]
	}

	hostname := strings.TrimPrefix(selectedResource.URL, "https://")

	// Store tokens
	if err := auth.StoreToken(hostname, tokens); err != nil {
		return fmt.Errorf("failed to store tokens: %w", err)
	}

	// Update config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg.SetHost(hostname, &config.HostConfig{
		Hostname: hostname,
		CloudID:  selectedResource.ID,
	})
	cfg.CurrentHost = hostname

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, output.Success.Render("Authentication successful!"))
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintf(opts.IO.Out, "Logged in to: %s\n", hostname)
	fmt.Fprintf(opts.IO.Out, "Cloud ID: %s\n", selectedResource.ID)

	return nil
}
