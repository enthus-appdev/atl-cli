package auth

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTokenSetIsExpired tests the IsExpired method.
func TestTokenSetIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired (1 hour from now)",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired (1 hour ago)",
			expiresAt: time.Now().Add(-time.Hour),
			want:      true,
		},
		{
			name:      "expires in 4 minutes (considered expired due to 5-min buffer)",
			expiresAt: time.Now().Add(4 * time.Minute),
			want:      true,
		},
		{
			name:      "expires in 6 minutes (not expired yet)",
			expiresAt: time.Now().Add(6 * time.Minute),
			want:      false,
		},
		{
			name:      "expires exactly now",
			expiresAt: time.Now(),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenSet := &TokenSet{
				AccessToken: "test-token",
				ExpiresAt:   tt.expiresAt,
			}

			got := tokenSet.IsExpired()
			if got != tt.want {
				t.Errorf("TokenSet.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTokenSetJSONSerialization tests JSON marshaling/unmarshaling of TokenSet.
func TestTokenSetJSONSerialization(t *testing.T) {
	original := &TokenSet{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		TokenType:    "Bearer",
		ExpiresAt:    time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
		Scopes:       []string{"read:jira-work", "write:jira-work"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded TokenSet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify fields
	if decoded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", decoded.AccessToken, original.AccessToken)
	}
	if decoded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", decoded.RefreshToken, original.RefreshToken)
	}
	if decoded.TokenType != original.TokenType {
		t.Errorf("TokenType = %q, want %q", decoded.TokenType, original.TokenType)
	}
	if !decoded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", decoded.ExpiresAt, original.ExpiresAt)
	}
	if len(decoded.Scopes) != len(original.Scopes) {
		t.Errorf("Scopes length = %d, want %d", len(decoded.Scopes), len(original.Scopes))
	}
}

// TestTokenSetJSONTags tests that JSON tags are correctly applied.
func TestTokenSetJSONTags(t *testing.T) {
	tokenSet := &TokenSet{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(tokenSet)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	// Verify JSON field names use snake_case
	expectedFields := []string{
		`"access_token"`,
		`"refresh_token"`,
		`"token_type"`,
		`"expires_at"`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain %s, got: %s", field, jsonStr)
		}
	}
}

// TestTokenSetEmptyScopes tests TokenSet with empty/nil scopes.
func TestTokenSetEmptyScopes(t *testing.T) {
	// Test with nil scopes
	tokenSet := &TokenSet{
		AccessToken: "token",
		Scopes:      nil,
	}

	data, err := json.Marshal(tokenSet)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded TokenSet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Scopes should be nil or empty after round-trip
	if len(decoded.Scopes) > 0 {
		t.Errorf("Scopes should be nil or empty, got %v", decoded.Scopes)
	}
}

// TestKeyringServiceConstant tests that the keyring service name is set correctly.
func TestKeyringServiceConstant(t *testing.T) {
	if KeyringService != "atlassian-cli" {
		t.Errorf("KeyringService = %q, want %q", KeyringService, "atlassian-cli")
	}
}

// TestTokenSetIsExpiredEdgeCases tests edge cases for token expiration.
func TestTokenSetIsExpiredEdgeCases(t *testing.T) {
	// Test with zero time (should be expired)
	zeroTime := &TokenSet{
		AccessToken: "token",
		ExpiresAt:   time.Time{},
	}
	if !zeroTime.IsExpired() {
		t.Error("TokenSet with zero ExpiresAt should be considered expired")
	}

	// Test with very far future time
	farFuture := &TokenSet{
		AccessToken: "token",
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour), // 1 year
	}
	if farFuture.IsExpired() {
		t.Error("TokenSet with far future ExpiresAt should not be expired")
	}
}

// TestTokenSetWithAllFields tests TokenSet with all fields populated.
func TestTokenSetWithAllFields(t *testing.T) {
	tokenSet := &TokenSet{
		AccessToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		RefreshToken: "refresh_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes: []string{
			"read:jira-work",
			"write:jira-work",
			"read:jira-user",
			"read:confluence-content.all",
			"write:confluence-content",
			"offline_access",
		},
	}

	// Verify all fields are accessible
	if tokenSet.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if tokenSet.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if tokenSet.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", tokenSet.TokenType, "Bearer")
	}
	if len(tokenSet.Scopes) != 6 {
		t.Errorf("Scopes count = %d, want 6", len(tokenSet.Scopes))
	}
	if tokenSet.IsExpired() {
		t.Error("Token with 1 hour expiry should not be expired")
	}
}

// helper function to check string containment
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestStoreAndGetToken tests file-based token storage and retrieval.
func TestStoreAndGetToken(t *testing.T) {
	// Use a unique hostname to avoid conflicts
	hostname := "test-store-get.atlassian.net"

	// Clean up before and after test
	defer DeleteToken(hostname)
	DeleteToken(hostname)

	// Create test tokens
	original := &TokenSet{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		Scopes:       []string{"read:jira-work", "write:jira-work"},
	}

	// Store tokens
	if err := StoreToken(hostname, original); err != nil {
		t.Fatalf("StoreToken() error = %v", err)
	}

	// Retrieve tokens
	retrieved, err := GetToken(hostname)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetToken() returned nil")
	}

	// Verify fields
	if retrieved.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", retrieved.AccessToken, original.AccessToken)
	}
	if retrieved.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", retrieved.RefreshToken, original.RefreshToken)
	}
	if retrieved.TokenType != original.TokenType {
		t.Errorf("TokenType = %q, want %q", retrieved.TokenType, original.TokenType)
	}
	if !retrieved.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", retrieved.ExpiresAt, original.ExpiresAt)
	}
	if len(retrieved.Scopes) != len(original.Scopes) {
		t.Errorf("Scopes length = %d, want %d", len(retrieved.Scopes), len(original.Scopes))
	}
}

// TestGetTokenNotExists tests GetToken for non-existent hostname.
func TestGetTokenNotExists(t *testing.T) {
	hostname := "nonexistent-host.atlassian.net"

	// Ensure it doesn't exist
	DeleteToken(hostname)

	// Get should return nil, nil
	tokens, err := GetToken(hostname)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if tokens != nil {
		t.Errorf("GetToken() = %v, want nil", tokens)
	}
}

// TestDeleteToken tests token deletion.
func TestDeleteToken(t *testing.T) {
	hostname := "test-delete.atlassian.net"

	// Store a token first
	tokens := &TokenSet{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	if err := StoreToken(hostname, tokens); err != nil {
		t.Fatalf("StoreToken() error = %v", err)
	}

	// Delete it
	if err := DeleteToken(hostname); err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}

	// Verify it's gone
	retrieved, err := GetToken(hostname)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if retrieved != nil {
		t.Error("Token should be deleted")
	}
}

// TestDeleteTokenNotExists tests deleting non-existent token.
func TestDeleteTokenNotExists(t *testing.T) {
	hostname := "nonexistent-delete.atlassian.net"

	// Delete should succeed (no error) even if not exists
	if err := DeleteToken(hostname); err != nil {
		t.Errorf("DeleteToken() error = %v, want nil", err)
	}
}

// TestListStoredHosts tests listing stored hosts.
func TestListStoredHosts(t *testing.T) {
	// Create test tokens for multiple hosts
	hosts := []string{
		"list-test-1.atlassian.net",
		"list-test-2.atlassian.net",
	}

	// Clean up before and after
	for _, h := range hosts {
		defer DeleteToken(h)
		DeleteToken(h)
	}

	// Store tokens for each host
	for _, h := range hosts {
		tokens := &TokenSet{
			AccessToken: "test-token-" + h,
			ExpiresAt:   time.Now().Add(time.Hour),
		}
		if err := StoreToken(h, tokens); err != nil {
			t.Fatalf("StoreToken(%s) error = %v", h, err)
		}
	}

	// List hosts
	storedHosts, err := ListStoredHosts()
	if err != nil {
		t.Fatalf("ListStoredHosts() error = %v", err)
	}

	// Verify our test hosts are in the list
	for _, expected := range hosts {
		found := false
		for _, stored := range storedHosts {
			if stored == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListStoredHosts() missing %q", expected)
		}
	}
}

// TestTokenFilePathSanitization tests hostname sanitization for file paths.
func TestTokenFilePathSanitization(t *testing.T) {
	// Test that special characters are handled
	hostname := "test:special/chars\\host.atlassian.net"

	tokens := &TokenSet{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	// Should not fail with special characters
	if err := StoreToken(hostname, tokens); err != nil {
		t.Fatalf("StoreToken() with special chars error = %v", err)
	}

	// Clean up
	defer DeleteToken(hostname)

	// Should be retrievable
	retrieved, err := GetToken(hostname)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if retrieved == nil {
		t.Error("GetToken() returned nil")
	}
}
