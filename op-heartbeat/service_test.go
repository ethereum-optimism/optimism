package op_heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	httpPort := freePort(t)
	metricsPort := freePort(t)
	cfg := Config{
		HTTPAddr: "127.0.0.1",
		HTTPPort: httpPort,
		Metrics: opmetrics.CLIConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1",
			ListenPort: metricsPort,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	exitC := make(chan error, 1)
	go func() {
		exitC <- Start(ctx, log.New(), cfg, "foobar")
	}()

	// Make sure that the service properly starts
	select {
	case <-time.NewTimer(100 * time.Millisecond).C:
		// pass
	case err := <-exitC:
		t.Fatalf("unexpected error on startup: %v", err)
	}

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
			req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://127.0.0.1:%d", httpPort), bytes.NewReader(data))
			require.NoError(t, err)
			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, res.StatusCode, 204)

			metricsRes, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", metricsPort))
			require.NoError(t, err)
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

func freePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)
	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
