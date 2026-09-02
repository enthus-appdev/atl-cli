// Package doctor implements `atl doctor`, a diagnostic command that checks
// auth and config health without touching the network.
package doctor

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// Options holds the options for the doctor command.
type Options struct {
	IO   *iostreams.IOStreams
	JSON bool
}

// NewCmdDoctor creates the doctor command.
func NewCmdDoctor(ios *iostreams.IOStreams) *cobra.Command {
	opts := &Options{IO: ios}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose auth and config health",
		Long: `Check the health of your atl setup: config file, OAuth app credentials
(and which layer supplies them), and per-host token state.

Credential values are shown partially masked so you can cross-check them
against your OAuth app in the Atlassian developer console. The client secret
is never shown beyond its last 4 characters.`,
		Example: `  atl doctor
  atl doctor --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// Check is one diagnostic result.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", or "error"
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Report is the full doctor output.
type Report struct {
	ConfigFile       string  `json:"config_file"`
	CredentialSource string  `json:"credential_source,omitempty"`
	ClientIDMasked   string  `json:"client_id_masked,omitempty"`
	ClientSecretHint string  `json:"client_secret_hint,omitempty"`
	Checks           []Check `json:"checks"`
}

func runDoctor(opts *Options) error {
	report := &Report{ConfigFile: config.ConfigFile()}

	cfg := checkConfig(report)
	checkCredentials(report, cfg)
	checkHosts(report, cfg)

	if opts.JSON {
		return output.JSON(opts.IO.Out, report)
	}
	return renderText(opts, report)
}

func (r *Report) add(status, name, message, hint string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: status, Message: message, Hint: hint})
}

// checkConfig verifies the config file loads and is not world-readable.
// Returns the loaded config (never nil; falls back to an empty config so the
// remaining checks can still report what they know).
func checkConfig(r *Report) *config.Config {
	path := config.ConfigFile()
	info, statErr := os.Stat(path)
	switch {
	case os.IsNotExist(statErr):
		r.add("warn", "config file", fmt.Sprintf("%s does not exist", path),
			"Run 'atl auth login' to get started")
	case statErr != nil:
		r.add("error", "config file", fmt.Sprintf("cannot stat %s: %v", path, statErr), "")
	case info.Mode().Perm()&0o077 != 0:
		r.add("warn", "config file permissions",
			fmt.Sprintf("%s is readable by other users (mode %o)", path, info.Mode().Perm()),
			fmt.Sprintf("Run 'chmod 600 %s'", path))
	default:
		r.add("ok", "config file", fmt.Sprintf("%s (mode %o)", path, info.Mode().Perm()), "")
	}

	cfg, err := config.Load()
	if err != nil {
		r.add("error", "config parse", err.Error(), "")
		return &config.Config{}
	}
	return cfg
}

// checkCredentials reports which layer supplies the OAuth app credentials and
// shows them partially masked for cross-checking. The client ID is not a
// secret in OAuth 2.0 (it is visible in browser URLs during login), so first
// and last characters are shown. The client secret IS sensitive: only its
// length and last 4 characters are ever revealed.
func checkCredentials(r *Report, cfg *config.Config) {
	var configID, configSecret string
	if cfg.OAuth != nil {
		configID, configSecret = cfg.OAuth.ClientID, cfg.OAuth.ClientSecret
	}
	id, secret, source := auth.ResolveClientCredentials(configID, configSecret)

	// A lone env var half is silently skipped by the resolver — surface it,
	// since it usually means a typo'd or forgotten variable.
	envID, envSecret := os.Getenv("ATLASSIAN_CLIENT_ID"), os.Getenv("ATLASSIAN_CLIENT_SECRET")
	if (envID == "") != (envSecret == "") {
		r.add("warn", "environment variables",
			"only one of ATLASSIAN_CLIENT_ID / ATLASSIAN_CLIENT_SECRET is set; the environment layer is being ignored",
			"Set both variables, or unset the stray one")
	}

	if id == "" || secret == "" {
		r.add("error", "oauth credentials", "not configured in any layer",
			"Run 'atl auth set-credentials' (keychain) or set ATLASSIAN_CLIENT_ID/ATLASSIAN_CLIENT_SECRET")
		return
	}

	r.CredentialSource = string(source)
	r.ClientIDMasked = maskID(id)
	r.ClientSecretHint = secretHint(secret)

	msg := fmt.Sprintf("from %s — client ID %s, client secret %s", source, r.ClientIDMasked, r.ClientSecretHint)
	if source == auth.SourceConfig {
		r.add("warn", "oauth credentials", msg,
			"Credentials sit in plaintext in config.yaml; prefer 'atl auth set-credentials' to move them into the OS keychain")
	} else {
		r.add("ok", "oauth credentials", msg, "")
	}

	// A complete pair in a shadowed lower layer diverging from the active one
	// is a classic "why is my login using the wrong app" trap.
	if source != auth.SourceConfig && configID != "" && configSecret != "" && configID != id {
		r.add("warn", "shadowed credentials",
			fmt.Sprintf("config file holds a DIFFERENT client ID (%s) than the active %s layer", maskID(configID), source),
			"Remove the stale pair from config.yaml if it is no longer used")
	}
	if kc, ok := auth.GetClientCredentials(); ok && source == auth.SourceEnv && kc.ClientID != id {
		r.add("warn", "shadowed credentials",
			fmt.Sprintf("OS keychain holds a DIFFERENT client ID (%s) than the environment variables", maskID(kc.ClientID)),
			"Unset the env vars or update the keychain entry so they agree")
	}
}

// checkHosts verifies host configuration and per-host token state.
func checkHosts(r *Report, cfg *config.Config) {
	if len(cfg.Hosts) == 0 {
		r.add("warn", "hosts", "no hosts configured", "Run 'atl auth login'")
		return
	}

	activeHost := cfg.InvocationHost()
	if activeHost == "" {
		r.add("warn", "current host", "no current host set",
			"Run 'atl config use-context <host>'")
	} else if cfg.GetHost(activeHost) == nil {
		r.add("error", "current host",
			fmt.Sprintf("current host %q is not in the hosts list", activeHost),
			"Run 'atl config use-context <host>' with a configured host")
	} else {
		r.add("ok", "current host", activeHost, "")
	}

	for hostname, hostCfg := range cfg.Hosts {
		name := "token: " + hostname
		if hostCfg.CloudID == "" {
			r.add("warn", "host: "+hostname, "missing cloud ID", "Re-run 'atl auth login'")
		}
		tokens, err := auth.GetToken(hostname)
		switch {
		case err != nil:
			r.add("error", name, fmt.Sprintf("cannot read token: %v", err), "")
		case tokens == nil:
			r.add("error", name, "no token stored", "Run 'atl auth login'")
		case tokens.RefreshToken == "" && tokens.IsExpired():
			r.add("error", name, "token expired and no refresh token stored", "Run 'atl auth login'")
		case tokens.IsExpired():
			r.add("warn", name, fmt.Sprintf("token expired at %s (will auto-refresh on next use)",
				tokens.ExpiresAt.Format(time.RFC3339)), "Or run 'atl auth refresh' now")
		default:
			r.add("ok", name, fmt.Sprintf("valid until %s", tokens.ExpiresAt.Format(time.RFC3339)), "")
		}
	}

	// Token files for hosts no longer in the config hold live refresh tokens
	// but are invisible to every other command.
	stored, err := auth.ListStoredHosts()
	if err == nil {
		for _, h := range stored {
			if cfg.GetHost(h) == nil {
				r.add("warn", "orphaned token: "+h, "token file exists but host is not configured",
					fmt.Sprintf("Run 'atl auth logout --hostname %s' to remove it", h))
			}
		}
	}
}

// maskID shows enough of a client ID to identify it without printing it whole.
func maskID(s string) string {
	if len(s) < 10 {
		return "****"
	}
	return s[:4] + "…" + s[len(s)-4:]
}

// secretHint identifies a secret by length and last 4 characters only —
// enough to cross-check against the developer console, useless to reconstruct.
func secretHint(s string) string {
	if len(s) < 8 {
		return fmt.Sprintf("(%d chars)", len(s))
	}
	return fmt.Sprintf("(%d chars, ends …%s)", len(s), s[len(s)-4:])
}

func renderText(opts *Options, r *Report) error {
	symbols := map[string]string{
		"ok":    output.Success.Render("✓"),
		"warn":  output.Warning.Render("!"),
		"error": output.Error.Render("✗"),
	}

	var warns, errs int
	for _, c := range r.Checks {
		fmt.Fprintf(opts.IO.Out, "%s %s: %s\n", symbols[c.Status], output.Bold.Render(c.Name), c.Message)
		if c.Hint != "" {
			fmt.Fprintf(opts.IO.Out, "    %s\n", c.Hint)
		}
		switch c.Status {
		case "warn":
			warns++
		case "error":
			errs++
		}
	}

	fmt.Fprintln(opts.IO.Out)
	switch {
	case errs > 0:
		fmt.Fprintf(opts.IO.Out, "%s\n", output.Error.Render(fmt.Sprintf("%d error(s), %d warning(s)", errs, warns)))
		return fmt.Errorf("doctor found %d error(s)", errs)
	case warns > 0:
		fmt.Fprintf(opts.IO.Out, "%s\n", output.Warning.Render(fmt.Sprintf("%d warning(s)", warns)))
	default:
		fmt.Fprintf(opts.IO.Out, "%s\n", output.Success.Render("All checks passed"))
	}
	return nil
}
