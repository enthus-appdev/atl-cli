package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
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
	// Load config for OAuth credentials and API version
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve OAuth credentials: env vars > OS keychain > config file
	clientID, clientSecret, _, err := requireClientCredentials(cfg)
	if err != nil {
		return err
	}

	// Get default scopes (granular Confluence + classic Jira)
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

	// Normalize hostname (strip https:// prefix if user pasted a URL)
	if opts.Hostname != "" {
		opts.Hostname = config.NormalizeHostname(opts.Hostname)
	}

	// Select the site to authenticate against. The OAuth callback never carries
	// the site chosen in the browser consent screen — the token is account-wide
	// and accessible-resources lists every site the account can reach. So when
	// the target is ambiguous we ask rather than defaulting to an arbitrary
	// resources[0], which silently lands on the wrong site.
	var selectedResource *api.AccessibleResource
	switch {
	case opts.Hostname != "":
		for _, r := range resources {
			if strings.TrimPrefix(r.URL, "https://") == opts.Hostname {
				selectedResource = r
				break
			}
		}
		if selectedResource == nil {
			return fmt.Errorf("site %s not found in accessible resources", opts.Hostname)
		}
	case len(resources) == 1:
		selectedResource = resources[0]
	default:
		selectedResource, err = selectResource(opts.IO, resources)
		if err != nil {
			return err
		}
	}

	hostname := strings.TrimPrefix(selectedResource.URL, "https://")

	// Store tokens
	if err := auth.StoreToken(hostname, tokens); err != nil {
		return fmt.Errorf("failed to store tokens: %w", err)
	}

	// Update config with host info
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

// selectResource asks the user which accessible site to authenticate against.
// Non-interactive stdin errors out with guidance rather than guessing, so a
// scripted login can never silently land on the wrong site.
func selectResource(io *iostreams.IOStreams, resources []*api.AccessibleResource) (*api.AccessibleResource, error) {
	if !io.IsStdinTTY {
		return nil, fmt.Errorf("account can access %d Atlassian sites; re-run with --hostname to choose one (e.g. --hostname %s)",
			len(resources), strings.TrimPrefix(resources[0].URL, "https://"))
	}

	options := make([]string, len(resources))
	for i, r := range resources {
		options[i] = fmt.Sprintf("%s (%s)", strings.TrimPrefix(r.URL, "https://"), r.Name)
	}

	var idx int
	prompt := &survey.Select{
		Message: "Select the Atlassian site to log in to:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &idx); err != nil {
		return nil, fmt.Errorf("site selection canceled: %w", err)
	}

	return resources[idx], nil
}
