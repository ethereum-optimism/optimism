package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// JSON API health status values returned by rollup-boost
const (
	jsonHealthStatusHealthy   = "Healthy"
	jsonHealthStatusPartial   = "PartialContent"
	jsonHealthStatusUnhealthy = "ServiceUnavailable"
)

// RollupBoostNextClient retrieves rollup-boost health using the JSON-based healthcheck endpoint.
type RollupBoostNextClient struct {
	url        string
	httpClient *http.Client
}

// RollupBoostNextHealthResponse captures the JSON payload returned by the rollup-boost health endpoint.
type RollupBoostNextHealthResponse struct {
	Version           string `json:"version"`
	RollupBoostHealth string `json:"rollup_boost_health"`
}

// NewRollupBoostNextClient constructs a client for querying the rollup-boost health endpoint.
// The url parameter should be the full URL including path (e.g., "http://localhost:8080/healthz").
func NewRollupBoostNextClient(url string, httpClient *http.Client) *RollupBoostNextClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RollupBoostNextClient{
		url:        url,
		httpClient: httpClient,
	}
}

// Healthcheck fetches the rollup-boost health endpoint and interprets the JSON payload.
func (c *RollupBoostNextClient) Healthcheck(ctx context.Context) (HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RollupBoostNextHealthResponse
	// Limit response size to 1 MiB to prevent memory exhaustion from malicious servers
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Map JSON API values to internal constants
	switch payload.RollupBoostHealth {
	case jsonHealthStatusHealthy:
		return HealthStatusHealthy, nil
	case jsonHealthStatusPartial:
		return HealthStatusPartial, nil
	case jsonHealthStatusUnhealthy:
		return HealthStatusUnhealthy, nil
	default:
		return "", fmt.Errorf("unexpected rollup_boost_health: %q", payload.RollupBoostHealth)
	}
}

// Ensure RollupBoostNextClient implements RollupBoostHealthChecker
var _ RollupBoostHealthChecker = (*RollupBoostNextClient)(nil)
