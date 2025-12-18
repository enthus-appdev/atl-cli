// Package config provides configuration management for the Atlassian CLI.
//
// Configuration is stored in YAML format at ~/.config/atlassian/config.yaml
// (following XDG Base Directory Specification). The location can be overridden
// using the ATLASSIAN_CONFIG_DIR environment variable.
//
// The configuration includes:
//   - OAuth credentials for authentication
//   - Per-host settings (cloud ID, default project, etc.)
//   - User preferences (editor, pager, output format)
//   - Command aliases
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
// It holds all user settings, host configurations, and OAuth credentials.
type Config struct {
	Version             int                    `yaml:"version"`
	CurrentHost         string                 `yaml:"current_host,omitempty"`
	Hosts               map[string]*HostConfig `yaml:"hosts,omitempty"`
	Aliases             map[string]string      `yaml:"aliases,omitempty"`
	DefaultOutputFormat string                 `yaml:"default_output_format,omitempty"`
	Editor              string                 `yaml:"editor,omitempty"`
	Pager               string                 `yaml:"pager,omitempty"`
	OAuth               *OAuthConfig           `yaml:"oauth,omitempty"`
}

// OAuthConfig holds OAuth 2.0 application credentials.
// These are obtained by creating an OAuth app at https://developer.atlassian.com/console/myapps/
// and are used to authenticate users via the OAuth 2.0 authorization code flow.
type OAuthConfig struct {
	ClientID     string `yaml:"client_id"`     // OAuth app client ID
	ClientSecret string `yaml:"client_secret"` // OAuth app client secret
}

// HostConfig represents configuration for a specific Atlassian cloud instance.
// Each host corresponds to a unique Atlassian site (e.g., mycompany.atlassian.net).
type HostConfig struct {
	Hostname       string `yaml:"hostname"`                  // The Atlassian site hostname (e.g., "mycompany.atlassian.net")
	CloudID        string `yaml:"cloud_id,omitempty"`        // Unique cloud instance identifier from Atlassian API
	User           string `yaml:"user,omitempty"`            // Authenticated user's email or display name
	Protocol       string `yaml:"protocol,omitempty"`        // Protocol to use (defaults to "https")
	OAuthAppID     string `yaml:"oauth_app_id,omitempty"`    // OAuth app ID used for this host
	DefaultProject string `yaml:"default_project,omitempty"` // Default Jira project key for commands
}

var (
	configDir  string
	configOnce sync.Once
)

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	configOnce.Do(func() {
		if dir := os.Getenv("ATLASSIAN_CONFIG_DIR"); dir != "" {
			configDir = dir
			return
		}

		// Use XDG_CONFIG_HOME if set, otherwise use ~/.config
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			configDir = filepath.Join(xdgConfig, "atlassian")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				configDir = ".atlassian"
				return
			}
			configDir = filepath.Join(home, ".config", "atlassian")
		}
	})
	return configDir
}

// ConfigFile returns the path to the main configuration file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load reads the configuration from disk.
func Load() (*Config, error) {
	cfg := &Config{
		Version: 1,
		Hosts:   make(map[string]*HostConfig),
		Aliases: make(map[string]string),
	}

	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return default config if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(ConfigFile(), data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetHost returns the configuration for a specific host.
func (c *Config) GetHost(hostname string) *HostConfig {
	if c.Hosts == nil {
		return nil
	}
	return c.Hosts[hostname]
}

// SetHost sets the configuration for a specific host.
func (c *Config) SetHost(hostname string, host *HostConfig) {
	if c.Hosts == nil {
		c.Hosts = make(map[string]*HostConfig)
	}
	c.Hosts[hostname] = host
}

// RemoveHost removes a host from the configuration.
func (c *Config) RemoveHost(hostname string) {
	if c.Hosts != nil {
		delete(c.Hosts, hostname)
	}
	if c.CurrentHost == hostname {
		c.CurrentHost = ""
	}
}

// CurrentHostConfig returns the configuration for the current host.
func (c *Config) CurrentHostConfig() *HostConfig {
	if c.CurrentHost == "" {
		return nil
	}
	return c.GetHost(c.CurrentHost)
}

// Get returns a configuration value by key.
func (c *Config) Get(key string) string {
	switch key {
	case "current_host":
		return c.CurrentHost
	case "default_output_format":
		return c.DefaultOutputFormat
	case "editor":
		return c.Editor
	case "pager":
		return c.Pager
	default:
		return ""
	}
}

// Set sets a configuration value by key.
func (c *Config) Set(key, value string) error {
	switch key {
	case "current_host":
		c.CurrentHost = value
	case "default_output_format":
		c.DefaultOutputFormat = value
	case "editor":
		c.Editor = value
	case "pager":
		c.Pager = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}
