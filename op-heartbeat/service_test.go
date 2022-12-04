package op_heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestService(t *testing.T) {
	cfg := Config{
		HTTPAddr:        "127.0.0.1",
		HTTPPort:        8080,
		HTTPMaxBodySize: 1024 * 1024,
		Metrics: opmetrics.CLIConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1",
			ListenPort: 7300,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	exitC := make(chan error, 1)
	go func() {
		exitC <- Start(ctx, log.New(), cfg, "foobar")
	}()

	tests := []struct {
		name        string
		hb          heartbeat.Payload
		metricName  string
		metricValue int
	}{
		{
			"no whitelisted version",
			heartbeat.Payload{
				Version: "not_whitelisted",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 10,
			},
			`op_heartbeat_heartbeats{chain_id="10",version="unknown"}`,
			1,
		},
		{
			"no whitelisted chain",
			heartbeat.Payload{
				Version: "v0.1.0-beta.1",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 999,
			},
			`op_heartbeat_heartbeats{chain_id="unknown",version="v0.1.0-beta.1"}`,
			1,
		},
		{
			"both whitelisted",
			heartbeat.Payload{
				Version: "v0.1.0-beta.1",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 10,
			},
			`op_heartbeat_heartbeats{chain_id="10",version="v0.1.0-beta.1"}`,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.hb)
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:8080", bytes.NewReader(data))
			require.NoError(t, err)
			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.Equal(t, res.StatusCode, 204)

			metricsRes, err := http.Get("http://127.0.0.1:7300")
			defer metricsRes.Body.Close()
			require.NoError(t, err)
			metricsBody, err := io.ReadAll(metricsRes.Body)
			require.NoError(t, err)
			require.Contains(t, string(metricsBody), fmt.Sprintf("%s %d", tt.metricName, tt.metricValue))
		})
	}

	cancel()
	require.NoError(t, <-exitC)
}
