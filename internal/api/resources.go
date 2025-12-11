package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AccessibleResource represents an accessible Atlassian cloud resource.
type AccessibleResource struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	AvatarURL string   `json:"avatarUrl,omitempty"`
}

// GetAccessibleResources fetches the list of accessible Atlassian resources.
// This is needed to get the cloud ID for a site.
func GetAccessibleResources(ctx context.Context, accessToken string) ([]*AccessibleResource, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AtlassianAPIURL+"/oauth/token/accessible-resources", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var resources []*AccessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resources, nil
}
