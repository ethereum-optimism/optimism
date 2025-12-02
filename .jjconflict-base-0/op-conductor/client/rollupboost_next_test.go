package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollupBoostNextHealthcheck(t *testing.T) {
	testCases := []struct {
		name       string
		response   interface{}
		statusCode int
		wantStatus HealthStatus
		wantErr    string
	}{
		{
			name: "healthy",
			response: RollupBoostNextHealthResponse{
				Version:           "1.0.0",
				RollupBoostHealth: "Healthy", // JSON API value
			},
			statusCode: http.StatusOK,
			wantStatus: HealthStatusHealthy,
		},
		{
			name: "partial",
			response: RollupBoostNextHealthResponse{
				Version:           "1.0.0",
				RollupBoostHealth: "PartialContent", // JSON API value
			},
			statusCode: http.StatusOK,
			wantStatus: HealthStatusPartial,
		},
		{
			name: "unhealthy",
			response: RollupBoostNextHealthResponse{
				Version:           "1.0.0",
				RollupBoostHealth: "ServiceUnavailable", // JSON API value
			},
			statusCode: http.StatusOK,
			wantStatus: HealthStatusUnhealthy,
		},
		{
			name: "unexpected status code",
			response: RollupBoostNextHealthResponse{
				Version:           "1.0.0",
				RollupBoostHealth: "Healthy", // JSON API value
			},
			statusCode: http.StatusAccepted,
			wantErr:    "unexpected status code: 202",
		},
		{
			name:       "malformed json",
			response:   "{not-json",
			statusCode: http.StatusOK,
			wantErr:    "failed to decode response",
		},
		{
			name: "unknown health",
			response: RollupBoostNextHealthResponse{
				Version:           "1.0.0",
				RollupBoostHealth: "Unknown",
			},
			statusCode: http.StatusOK,
			wantErr:    `unexpected rollup_boost_health: "Unknown"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, HealthzEndpoint, r.URL.Path)
				w.WriteHeader(tc.statusCode)

				switch v := tc.response.(type) {
				case string:
					_, _ = w.Write([]byte(v))
				default:
					require.NoError(t, json.NewEncoder(w).Encode(v))
				}
			}))
			defer server.Close()

			// Pass full URL including path
			client := NewRollupBoostNextClient(server.URL+HealthzEndpoint, server.Client())
			status, err := client.Healthcheck(context.Background())

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, status)
		})
	}
}
