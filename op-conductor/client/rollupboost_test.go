package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollupBoostHealthcheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       HealthStatus
		wantErr    bool
	}{
		{
			name:       "healthy response",
			statusCode: http.StatusOK,
			want:       HealthStatusHealthy,
			wantErr:    false,
		},
		{
			name:       "partial health response",
			statusCode: http.StatusPartialContent,
			want:       HealthStatusPartial,
			wantErr:    false,
		},
		{
			name:       "unhealthy response",
			statusCode: http.StatusServiceUnavailable,
			want:       HealthStatusUnhealthy,
			wantErr:    false,
		},
		{
			name:       "unexpected status code",
			statusCode: http.StatusBadRequest,
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns the desired status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, HealthzEndpoint, r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			// Create client that points to our test server
			client := NewRollupBoostClient(server.URL, server.Client())

			// Test the healthcheck
			got, err := client.Healthcheck(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
