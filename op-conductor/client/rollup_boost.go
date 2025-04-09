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

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusPartial   HealthStatus = "partial"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

type RollupBoostClient interface {
	Healthcheck(ctx context.Context) (HealthStatus, error)
}

type rollupBoostClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRollupBoostClient(baseURL string, httpClient *http.Client) RollupBoostClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &rollupBoostClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *rollupBoostClient) Healthcheck(ctx context.Context) (HealthStatus, error) {
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
