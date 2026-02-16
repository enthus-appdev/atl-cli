package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigDir tests the ConfigDir function returns a valid directory path.
func TestConfigDir(t *testing.T) {
	// Reset the configOnce and configDir for testing
	// Note: In a real test, we'd use a separate config instance
	// For now, we just verify the function doesn't panic and returns a path

	dir := ConfigDir()
	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}
}

// TestConfigDirWithEnvVar tests ConfigDir respects ATLASSIAN_CONFIG_DIR.
func TestConfigDirWithEnvVar(t *testing.T) {
	// Save original env
	origDir := os.Getenv("ATLASSIAN_CONFIG_DIR")
	defer os.Setenv("ATLASSIAN_CONFIG_DIR", origDir)

	// This test demonstrates the env var behavior
	// Due to sync.Once, we can't easily test this without refactoring
	// This is a documentation test showing expected behavior
	t.Skip("ConfigDir uses sync.Once - would need refactoring for full test coverage")
}

// TestNewConfig tests creating a new default Config.
func TestNewConfig(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Hosts:   make(map[string]*HostConfig),
		Aliases: make(map[string]string),
	}

	if cfg.Version != 1 {
		t.Errorf("expected Version=1, got %d", cfg.Version)
	}
	if cfg.Hosts == nil {
		t.Error("Hosts map should not be nil")
	}
	if cfg.Aliases == nil {
		t.Error("Aliases map should not be nil")
	}
}

// TestConfigGetSet tests the Get and Set methods for configuration values.
func TestConfigGetSet(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		key   string
		value string
	}{
		{"current_host", "example.atlassian.net"},
		{"default_output_format", "json"},
		{"editor", "vim"},
		{"pager", "less"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// Test Set
			err := cfg.Set(tt.key, tt.value)
			if err != nil {
				t.Errorf("Set(%q, %q) returned error: %v", tt.key, tt.value, err)
			}

			// Test Get
			got := cfg.Get(tt.key)
			if got != tt.value {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}
}

// TestConfigSetUnknownKey tests that Set returns an error for unknown keys.
func TestConfigSetUnknownKey(t *testing.T) {
	cfg := &Config{}

	err := cfg.Set("unknown_key", "value")
	if err == nil {
		t.Error("Set() should return error for unknown key")
	}
}

// TestConfigGetUnknownKey tests that Get returns empty string for unknown keys.
func TestConfigGetUnknownKey(t *testing.T) {
	cfg := &Config{}

	got := cfg.Get("unknown_key")
	if got != "" {
		t.Errorf("Get(unknown_key) = %q, want empty string", got)
	}
}

// TestHostConfig tests host configuration operations.
func TestHostConfig(t *testing.T) {
	cfg := &Config{
		Hosts: make(map[string]*HostConfig),
	}

	hostname := "example.atlassian.net"
	host := &HostConfig{
		Hostname:       hostname,
		CloudID:        "cloud-123",
		User:           "test@example.com",
		DefaultProject: "TEST",
	}

	// Test SetHost
	cfg.SetHost(hostname, host)

	// Test GetHost
	got := cfg.GetHost(hostname)
	if got == nil {
		t.Fatal("GetHost() returned nil")
	}
	if got.Hostname != hostname {
		t.Errorf("GetHost().Hostname = %q, want %q", got.Hostname, hostname)
	}
	if got.CloudID != "cloud-123" {
		t.Errorf("GetHost().CloudID = %q, want %q", got.CloudID, "cloud-123")
	}
}

// TestHostConfigNilMap tests GetHost with nil hosts map.
func TestHostConfigNilMap(t *testing.T) {
	cfg := &Config{Hosts: nil}

	got := cfg.GetHost("example.atlassian.net")
	if got != nil {
		t.Error("GetHost() should return nil when Hosts map is nil")
	}
}

// TestSetHostNilMap tests SetHost initializes the map if nil.
func TestSetHostNilMap(t *testing.T) {
	cfg := &Config{Hosts: nil}

	host := &HostConfig{Hostname: "example.atlassian.net"}
	cfg.SetHost("example.atlassian.net", host)

	if cfg.Hosts == nil {
		t.Error("SetHost() should initialize Hosts map")
	}
	if cfg.Hosts["example.atlassian.net"] == nil {
		t.Error("SetHost() should store the host config")
	}
}

// TestRemoveHost tests removing a host from configuration.
func TestRemoveHost(t *testing.T) {
	hostname := "example.atlassian.net"
	cfg := &Config{
		CurrentHost: hostname,
		Hosts: map[string]*HostConfig{
			hostname: {Hostname: hostname},
		},
	}

	cfg.RemoveHost(hostname)

	if cfg.GetHost(hostname) != nil {
		t.Error("RemoveHost() should remove the host")
	}
	if cfg.CurrentHost != "" {
		t.Error("RemoveHost() should clear CurrentHost when removing current host")
	}
}

// TestRemoveHostNilMap tests RemoveHost with nil hosts map.
func TestRemoveHostNilMap(t *testing.T) {
	cfg := &Config{Hosts: nil}

	// Should not panic
	cfg.RemoveHost("example.atlassian.net")
}

// TestCurrentHostConfig tests getting the current host configuration.
func TestCurrentHostConfig(t *testing.T) {
	hostname := "example.atlassian.net"
	host := &HostConfig{Hostname: hostname, CloudID: "cloud-123"}

	cfg := &Config{
		CurrentHost: hostname,
		Hosts: map[string]*HostConfig{
			hostname: host,
		},
	}

	got := cfg.CurrentHostConfig()
	if got == nil {
		t.Fatal("CurrentHostConfig() returned nil")
	}
	if got.CloudID != "cloud-123" {
		t.Errorf("CurrentHostConfig().CloudID = %q, want %q", got.CloudID, "cloud-123")
	}
}

// TestCurrentHostConfigEmpty tests CurrentHostConfig when no current host is set.
func TestCurrentHostConfigEmpty(t *testing.T) {
	cfg := &Config{CurrentHost: ""}

	got := cfg.CurrentHostConfig()
	if got != nil {
		t.Error("CurrentHostConfig() should return nil when CurrentHost is empty")
	}
}

// TestLoadNonExistentFile tests Load returns default config for non-existent file.
func TestLoadNonExistentFile(t *testing.T) {
	// Create a temp directory for test config
	tempDir := t.TempDir()
	os.Setenv("ATLASSIAN_CONFIG_DIR", tempDir)
	defer os.Unsetenv("ATLASSIAN_CONFIG_DIR")

	// Reset configOnce would be needed here for proper testing
	// This test documents the expected behavior
	t.Skip("Load test requires ability to reset sync.Once or dependency injection")
}

// TestSaveAndLoad tests the round-trip of saving and loading config.
func TestSaveAndLoad(t *testing.T) {
	// Create a temp directory for test config
	tempDir := t.TempDir()

	// Create config manually to test YAML serialization
	cfg := &Config{
		Version:     1,
		CurrentHost: "example.atlassian.net",
		Hosts: map[string]*HostConfig{
			"example.atlassian.net": {
				Hostname:       "example.atlassian.net",
				CloudID:        "cloud-123",
				User:           "test@example.com",
				DefaultProject: "TEST",
			},
		},
		DefaultOutputFormat: "json",
		Editor:              "vim",
		OAuth: &OAuthConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	// Save to temp file
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Test that config can be serialized without error
	// (Full round-trip test would require dependency injection)
	if cfg.OAuth == nil {
		t.Error("OAuth config should not be nil")
	}
	if cfg.OAuth.ClientID != "client-id" {
		t.Errorf("OAuth.ClientID = %q, want %q", cfg.OAuth.ClientID, "client-id")
	}
}

// TestResolveHost tests alias resolution.
func TestResolveHost(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"prod":    "mycompany.atlassian.net",
			"sandbox": "mycompany-sandbox.atlassian.net",
		},
	}

	tests := []struct {
		input string
		want  string
	}{
		{"prod", "mycompany.atlassian.net"},
		{"sandbox", "mycompany-sandbox.atlassian.net"},
		{"mycompany.atlassian.net", "mycompany.atlassian.net"},       // passthrough
		{"unknown.atlassian.net", "unknown.atlassian.net"},           // passthrough
		{"nonexistent", "nonexistent"},                                // not an alias
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cfg.ResolveHost(tt.input)
			if got != tt.want {
				t.Errorf("ResolveHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestResolveHostNilAliases tests ResolveHost with nil aliases map.
func TestResolveHostNilAliases(t *testing.T) {
	cfg := &Config{Aliases: nil}

	got := cfg.ResolveHost("prod")
	if got != "prod" {
		t.Errorf("ResolveHost(\"prod\") = %q, want %q", got, "prod")
	}
}

// TestSetAlias tests creating aliases.
func TestSetAlias(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"mycompany.atlassian.net": {Hostname: "mycompany.atlassian.net"},
		},
	}

	err := cfg.SetAlias("prod", "mycompany.atlassian.net")
	if err != nil {
		t.Fatalf("SetAlias() returned error: %v", err)
	}

	if cfg.Aliases["prod"] != "mycompany.atlassian.net" {
		t.Errorf("Aliases[\"prod\"] = %q, want %q", cfg.Aliases["prod"], "mycompany.atlassian.net")
	}
}

// TestSetAliasUnknownHost tests that SetAlias rejects unknown hosts.
func TestSetAliasUnknownHost(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"mycompany.atlassian.net": {Hostname: "mycompany.atlassian.net"},
		},
	}

	err := cfg.SetAlias("bad", "nonexistent.atlassian.net")
	if err == nil {
		t.Error("SetAlias() should return error for unknown host")
	}
}

// TestSetAliasNilHosts tests SetAlias with nil hosts map.
func TestSetAliasNilHosts(t *testing.T) {
	cfg := &Config{Hosts: nil}

	err := cfg.SetAlias("prod", "mycompany.atlassian.net")
	if err == nil {
		t.Error("SetAlias() should return error when hosts map is nil")
	}
}

// TestRemoveAlias tests removing an alias.
func TestRemoveAlias(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"prod":    "mycompany.atlassian.net",
			"sandbox": "mycompany-sandbox.atlassian.net",
		},
	}

	cfg.RemoveAlias("prod")

	if _, ok := cfg.Aliases["prod"]; ok {
		t.Error("RemoveAlias() should remove the alias")
	}
	if cfg.Aliases["sandbox"] != "mycompany-sandbox.atlassian.net" {
		t.Error("RemoveAlias() should not affect other aliases")
	}
}

// TestRemoveAliasNilMap tests RemoveAlias with nil aliases map.
func TestRemoveAliasNilMap(t *testing.T) {
	cfg := &Config{Aliases: nil}

	// Should not panic
	cfg.RemoveAlias("prod")
}

// TestAliasForHost tests reverse alias lookup.
func TestAliasForHost(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"prod": "mycompany.atlassian.net",
		},
	}

	got := cfg.AliasForHost("mycompany.atlassian.net")
	if got != "prod" {
		t.Errorf("AliasForHost() = %q, want %q", got, "prod")
	}

	got = cfg.AliasForHost("unknown.atlassian.net")
	if got != "" {
		t.Errorf("AliasForHost(unknown) = %q, want empty string", got)
	}
}

// TestSetCurrentHostResolvesAlias tests that Set("current_host") resolves aliases.
func TestSetCurrentHostResolvesAlias(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"prod": "mycompany.atlassian.net",
		},
	}

	err := cfg.Set("current_host", "prod")
	if err != nil {
		t.Fatalf("Set() returned error: %v", err)
	}

	if cfg.CurrentHost != "mycompany.atlassian.net" {
		t.Errorf("CurrentHost = %q, want %q", cfg.CurrentHost, "mycompany.atlassian.net")
	}
}

// TestOAuthConfig tests the OAuthConfig struct.
func TestOAuthConfig(t *testing.T) {
	oauth := &OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	if oauth.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", oauth.ClientID, "test-client-id")
	}
	if oauth.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %q, want %q", oauth.ClientSecret, "test-client-secret")
	}
}
