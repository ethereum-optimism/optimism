package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	// HealthzEndpoint is the fixed path for health checks
	HealthzEndpoint = "/healthz"
)

// HealthStatus represents the health state of rollup-boost.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusPartial   HealthStatus = "partial"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// RollupBoostHealthChecker is the common interface for rollup-boost health checking.
// Both RollupBoostClient and RollupBoostNextClient implement this interface.
type RollupBoostHealthChecker interface {
	Healthcheck(ctx context.Context) (HealthStatus, error)
}

// RollupBoostClient uses HTTP status codes to determine rollup-boost health.
type RollupBoostClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRollupBoostClient creates a client that interprets HTTP status codes for health.
func NewRollupBoostClient(baseURL string, httpClient *http.Client) *RollupBoostClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RollupBoostClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// Healthcheck returns health status based on HTTP status codes:
// 200 OK = Healthy, 206 Partial Content = Partial, 503 Service Unavailable = Unhealthy
func (c *RollupBoostClient) Healthcheck(ctx context.Context) (HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+HealthzEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read and discard body to ensure connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK: // 200
		return HealthStatusHealthy, nil
	case http.StatusPartialContent: // 206
		return HealthStatusPartial, nil
	case http.StatusServiceUnavailable: // 503
		return HealthStatusUnhealthy, nil
	default:
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// Ensure RollupBoostClient implements RollupBoostHealthChecker
var _ RollupBoostHealthChecker = (*RollupBoostClient)(nil)
