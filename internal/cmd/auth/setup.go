package auth

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// SetupOptions holds the options for the setup command.
type SetupOptions struct {
	IO           *iostreams.IOStreams
	ClientID     string
	ClientSecret string
	APIVersion   string
	Interactive  bool
}

// NewCmdSetup creates the setup command.
func NewCmdSetup(ios *iostreams.IOStreams) *cobra.Command {
	opts := &SetupOptions{
		IO:          ios,
		Interactive: true,
	}

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure OAuth credentials for authentication",
		Long: `Set up OAuth credentials for authenticating with Atlassian.

This command guides you through creating an OAuth 2.0 app in Atlassian
and storing the credentials. You only need to run this once.

You'll choose between two API versions:
  v1: Uses classic scopes (recommended for broad access)
  v2: Uses granular scopes (required for Confluence v2 API)

The credentials are stored locally in ~/.config/atlassian/config.yaml`,
		Example: `  # Interactive setup (recommended)
  atl auth setup

  # Non-interactive setup with v1 API (classic scopes)
  atl auth setup --client-id YOUR_ID --client-secret YOUR_SECRET --api-version v1

  # Non-interactive setup with v2 API (granular scopes)
  atl auth setup --client-id YOUR_ID --client-secret YOUR_SECRET --api-version v2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ClientID != "" && opts.ClientSecret != "" {
				opts.Interactive = false
			}
			return runSetup(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().StringVar(&opts.APIVersion, "api-version", "", "API version (v1 or v2)")

	return cmd
}

func runSetup(opts *SetupOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if already configured
	if cfg.OAuth != nil && cfg.OAuth.ClientID != "" && opts.Interactive {
		fmt.Fprintln(opts.IO.Out, "OAuth credentials are already configured.")
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintf(opts.IO.Out, "Client ID: %s...%s\n",
			cfg.OAuth.ClientID[:min(8, len(cfg.OAuth.ClientID))],
			cfg.OAuth.ClientID[max(0, len(cfg.OAuth.ClientID)-4):])
		if cfg.OAuth.APIVersion != "" {
			fmt.Fprintf(opts.IO.Out, "API Version: %s\n", cfg.OAuth.APIVersion)
		}
		fmt.Fprintln(opts.IO.Out, "")

		reader := bufio.NewReader(opts.IO.In)
		fmt.Fprint(opts.IO.Out, "Do you want to reconfigure? [y/N]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Fprintln(opts.IO.Out, "Setup cancelled.")
			return nil
		}
		fmt.Fprintln(opts.IO.Out, "")
	}

	clientID := opts.ClientID
	clientSecret := opts.ClientSecret
	apiVersion := config.APIVersion(opts.APIVersion)

	if opts.Interactive {
		reader := bufio.NewReader(opts.IO.In)

		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Atlassian CLI - OAuth Setup"))
		fmt.Fprintln(opts.IO.Out, "")

		// Step 0: Choose API version
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Step 0: Choose API version"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  The CLI supports two API versions with different OAuth scopes:")
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  "+output.Bold.Render("v1")+" - Classic scopes "+output.Faint.Render("(recommended)"))
		fmt.Fprintln(opts.IO.Out, "      • Broader permissions with fewer scope entries")
		fmt.Fprintln(opts.IO.Out, "      • Uses Confluence REST API v1")
		fmt.Fprintln(opts.IO.Out, "      • Simpler to configure in Atlassian Developer Console")
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  "+output.Bold.Render("v2")+" - Granular scopes")
		fmt.Fprintln(opts.IO.Out, "      • Fine-grained permissions")
		fmt.Fprintln(opts.IO.Out, "      • Uses Confluence REST API v2")
		fmt.Fprintln(opts.IO.Out, "      • Requires selecting individual scopes")
		fmt.Fprintln(opts.IO.Out, "")

		for {
			fmt.Fprint(opts.IO.Out, "  Choose API version ["+output.Bold.Render("1")+"/2]: ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)

			if answer == "" || answer == "1" || answer == "v1" {
				apiVersion = config.APIVersionV1
				break
			} else if answer == "2" || answer == "v2" {
				apiVersion = config.APIVersionV2
				break
			}
			fmt.Fprintln(opts.IO.Out, "  Please enter '1' or '2'")
		}

		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintf(opts.IO.Out, "  Selected: %s\n", output.Cyan.Render(string(apiVersion)))
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprintln(opts.IO.Out, "  This will open the Atlassian Developer Console where you'll")
		fmt.Fprintln(opts.IO.Out, "  create an OAuth app (takes about 2 minutes).")
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprint(opts.IO.Out, "  Press "+output.Bold.Render("Enter")+" to open browser (or 'q' to quit): ")
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) == "q" {
			return nil
		}

		// Open browser to developer console
		auth.OpenBrowser("https://developer.atlassian.com/console/myapps/")

		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Step 1: Create the app"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  In the browser:")
		fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Create")+" → "+output.Bold.Render("OAuth 2.0 integration"))
		fmt.Fprintln(opts.IO.Out, "    • Name your app (e.g., \"atl CLI\")")
		fmt.Fprintln(opts.IO.Out, "    • Agree to terms and click "+output.Bold.Render("Create"))
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprint(opts.IO.Out, "  Press "+output.Bold.Render("Enter")+" when done: ")
		reader.ReadString('\n')

		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Step 2: Configure callback URL"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  In the browser:")
		fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Authorization")+" in the left menu")
		fmt.Fprintln(opts.IO.Out, "    • Next to \"OAuth 2.0 (3LO)\", click "+output.Bold.Render("Configure"))
		fmt.Fprintln(opts.IO.Out, "    • Enter this callback URL:")
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "      "+output.Cyan.Render("http://localhost:8085/callback"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Save changes"))
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprint(opts.IO.Out, "  Press "+output.Bold.Render("Enter")+" when done: ")
		reader.ReadString('\n')

		// Show appropriate scopes based on API version
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Step 3: Add permissions"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  In the browser:")
		fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Permissions")+" in the left menu")
		fmt.Fprintln(opts.IO.Out, "")

		if apiVersion == config.APIVersionV1 {
			// Classic scopes for v1
			fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Add")+" next to \"Jira API\"")
			fmt.Fprintln(opts.IO.Out, "    • In the "+output.Bold.Render("Classic Scopes")+" tab, enable:")
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:jira-work"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("write:jira-work"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:jira-user"))
			fmt.Fprintln(opts.IO.Out, "")
			fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Add")+" next to \"Confluence API\"")
			fmt.Fprintln(opts.IO.Out, "    • In the "+output.Bold.Render("Classic Scopes")+" tab, enable:")
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:confluence-content.all"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("write:confluence-content"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:confluence-space.summary"))
		} else {
			// Granular scopes for v2
			fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Add")+" next to \"Jira API\"")
			fmt.Fprintln(opts.IO.Out, "    • In the "+output.Bold.Render("Classic Scopes")+" tab, enable:")
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:jira-work"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("write:jira-work"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:jira-user"))
			fmt.Fprintln(opts.IO.Out, "")
			fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Add")+" next to \"Confluence API\"")
			fmt.Fprintln(opts.IO.Out, "    • Select the "+output.Bold.Render("Granular Scopes")+" tab")
			fmt.Fprintln(opts.IO.Out, "    • Enable these scopes:")
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:space:confluence"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:page:confluence"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("write:page:confluence"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("read:content:confluence"))
			fmt.Fprintln(opts.IO.Out, "        "+output.Faint.Render("write:content:confluence"))
		}
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprint(opts.IO.Out, "  Press "+output.Bold.Render("Enter")+" when done: ")
		reader.ReadString('\n')

		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, output.Bold.Render("  Step 4: Copy credentials"))
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "  In the browser:")
		fmt.Fprintln(opts.IO.Out, "    • Click "+output.Bold.Render("Settings")+" in the left menu")
		fmt.Fprintln(opts.IO.Out, "    • Find \"Client ID\" and \"Secret\" under Authentication details")
		fmt.Fprintln(opts.IO.Out, "")

		fmt.Fprint(opts.IO.Out, "  Paste your "+output.Bold.Render("Client ID")+": ")
		clientID, _ = reader.ReadString('\n')
		clientID = strings.TrimSpace(clientID)

		if clientID == "" {
			return fmt.Errorf("client ID is required")
		}

		fmt.Fprint(opts.IO.Out, "  Paste your "+output.Bold.Render("Secret")+":    ")
		clientSecret, _ = reader.ReadString('\n')
		clientSecret = strings.TrimSpace(clientSecret)

		if clientSecret == "" {
			return fmt.Errorf("client secret is required")
		}
	} else {
		// Non-interactive mode: validate and default API version
		if apiVersion == "" {
			apiVersion = config.APIVersionV1
		}
		if apiVersion != config.APIVersionV1 && apiVersion != config.APIVersionV2 {
			return fmt.Errorf("invalid API version: %s (must be 'v1' or 'v2')", apiVersion)
		}
	}

	// Save to config
	cfg.OAuth = &config.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		APIVersion:   apiVersion,
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, "  "+output.Success.Render("Setup complete!"))
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintf(opts.IO.Out, "  API Version: %s\n", output.Cyan.Render(string(apiVersion)))
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, "  Now run: "+output.Cyan.Render("atl auth login"))
	fmt.Fprintln(opts.IO.Out, "")

	return nil
}
