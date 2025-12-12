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
	if decoded.Scopes != nil && len(decoded.Scopes) > 0 {
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
