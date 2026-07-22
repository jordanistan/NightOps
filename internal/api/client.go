package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	syncbundle "github.com/jordanistan/nightops/internal/sync"
)

// Client is a small client for the versioned local NightOps API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthValue  string
}

// NewClient creates a client for a NightOps API listener.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), HTTPClient: http.DefaultClient}
}

// Status is the API capability response.
type Status struct {
	Service      string `json:"service"`
	APIVersion   int    `json:"api_version"`
	OfflineFirst bool   `json:"offline_first"`
	SyncEnabled  bool   `json:"sync_enabled"`
}

// Status retrieves API and sync capability information.
func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	if err := c.get(ctx, "/api/v1/status", &result); err != nil {
		return Status{}, err
	}
	return result, nil
}

// Missions retrieves archived missions from the local API.
func (c *Client) Missions(ctx context.Context) ([]Mission, error) {
	var result struct {
		Missions []Mission `json:"missions"`
	}
	if err := c.get(ctx, "/api/v1/missions", &result); err != nil {
		return nil, err
	}
	return result.Missions, nil
}

// Mission retrieves one archived mission by stable ID.
func (c *Client) Mission(ctx context.Context, id string) (Mission, error) {
	if strings.TrimSpace(id) == "" || strings.Contains(id, "/") {
		return Mission{}, fmt.Errorf("mission ID is required")
	}
	var result Mission
	if err := c.get(ctx, "/api/v1/missions/"+id, &result); err != nil {
		return Mission{}, err
	}
	return result, nil
}

// ExportSync retrieves the current versioned offline sync bundle.
func (c *Client) ExportSync(ctx context.Context) (syncbundle.Bundle, error) {
	var result syncbundle.Bundle
	if err := c.get(ctx, "/api/v1/sync", &result); err != nil {
		return syncbundle.Bundle{}, err
	}
	return result, nil
}

// ImportSync sends a versioned offline sync bundle for conflict-safe merge.
func (c *Client) ImportSync(ctx context.Context, bundle syncbundle.Bundle) (syncbundle.MergeReport, error) {
	var result syncbundle.MergeReport
	if err := c.request(ctx, http.MethodPost, "/api/v1/sync", bundle, &result); err != nil {
		return syncbundle.MergeReport{}, err
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, path string, destination any) error {
	return c.request(ctx, http.MethodGet, path, nil, destination)
}

func (c *Client) request(ctx context.Context, method, path string, body any, destination any) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode API request: %w", err)
		}
		reader = strings.NewReader(string(encoded))
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create API request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(c.AuthValue) != "" {
		request.Header.Set("Authorization", "Bearer "+c.AuthValue)
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("API request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var detail struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&detail)
		if detail.Error == "" {
			detail.Error = response.Status
		}
		return fmt.Errorf("API returned %s: %s", response.Status, detail.Error)
	}
	if destination == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<20)).Decode(destination); err != nil {
		return fmt.Errorf("decode API response: %w", err)
	}
	return nil
}
