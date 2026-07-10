package auth

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// SetCredentialsOptions holds the options for the set-credentials command.
type SetCredentialsOptions struct {
	IO           *iostreams.IOStreams
	ClientID     string
	ClientSecret string
	FromStdin    bool
	Delete       bool
}

// NewCmdSetCredentials creates the set-credentials command.
func NewCmdSetCredentials(ios *iostreams.IOStreams) *cobra.Command {
	opts := &SetCredentialsOptions{IO: ios}

	cmd := &cobra.Command{
		Use:   "set-credentials",
		Short: "Store OAuth app credentials in the OS keychain",
		Long: `Store the OAuth 2.0 app client ID and secret in your operating system's
keychain (Keychain on macOS, Secret Service on Linux, Credential Manager on
Windows).

This is the recommended place for a shared or organization OAuth app: the
secret stays out of plaintext config files, and 'atl auth login' and
'atl auth refresh' read it automatically. Credentials are resolved in the
order: environment variables, then OS keychain, then config file.

The secret can be piped via --from-stdin so it never appears in shell history
or the process list — for example straight from a secret manager.`,
		Example: `  # Pipe the secret from stdin so it never hits shell history (preferred)
  printf '%s' "$SECRET" | atl auth set-credentials --client-id abc123 --from-stdin

  # Feed the secret directly from a secret manager
  gcloud secrets versions access latest --secret=atl-oauth \
    | atl auth set-credentials --client-id abc123 --from-stdin

  # Provide both values inline
  atl auth set-credentials --client-id abc123 --client-secret s3cr3t

  # Remove stored credentials
  atl auth set-credentials --delete`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetCredentials(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "OAuth client secret (prefer --from-stdin)")
	cmd.Flags().BoolVar(&opts.FromStdin, "from-stdin", false, "Read the client secret from stdin")
	cmd.Flags().BoolVar(&opts.Delete, "delete", false, "Remove stored credentials from the keychain")

	return cmd
}

func runSetCredentials(opts *SetCredentialsOptions) error {
	if opts.Delete {
		if opts.ClientID != "" || opts.ClientSecret != "" || opts.FromStdin {
			return fmt.Errorf("--delete cannot be combined with --client-id, --client-secret, or --from-stdin")
		}
		if err := auth.DeleteClientCredentials(); err != nil {
			return err
		}
		fmt.Fprintln(opts.IO.Out, output.Success.Render("Removed stored OAuth credentials from the keychain."))
		return nil
	}

	clientID := strings.TrimSpace(opts.ClientID)
	clientSecret := strings.TrimSpace(opts.ClientSecret)

	// Validate flags before touching stdin so a missing --client-id fails fast
	// instead of blocking on a read that will be discarded anyway.
	if clientID == "" {
		return fmt.Errorf("--client-id is required")
	}
	if opts.FromStdin {
		if clientSecret != "" {
			return fmt.Errorf("--from-stdin cannot be combined with --client-secret")
		}
		// An OAuth secret is small, so a large stream is a mistake (or abuse).
		// Read one byte past the cap and reject rather than silently truncate,
		// which would store a corrupt secret that only fails much later.
		const maxSecretBytes = 64 * 1024
		data, err := io.ReadAll(io.LimitReader(opts.IO.In, maxSecretBytes+1))
		if err != nil {
			return fmt.Errorf("failed to read secret from stdin: %w", err)
		}
		if len(data) > maxSecretBytes {
			return fmt.Errorf("secret read from stdin exceeds %d bytes; that is not a valid OAuth secret", maxSecretBytes)
		}
		clientSecret = strings.TrimSpace(string(data))
	}
	if clientSecret == "" {
		return fmt.Errorf("client secret is required (pass --client-secret or --from-stdin)")
	}

	if err := auth.StoreClientCredentials(auth.ClientCredentials{ClientID: clientID, ClientSecret: clientSecret}); err != nil {
		return err
	}

	fmt.Fprintln(opts.IO.Out, output.Success.Render("Stored OAuth credentials in the keychain."))
	fmt.Fprintln(opts.IO.Out, "")
	fmt.Fprintln(opts.IO.Out, "Now run: "+output.Cyan.Render("atl auth login"))
	return nil
}
