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

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
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
	srv, err := Start(ctx, log.New(), cfg, "foobar")
	// Make sure that the service properly starts
	require.NoError(t, err)

	defer cancel()
	defer func() {
		require.NoError(t, srv.Stop(ctx), "close heartbeat server")
	}()

	tests := []struct {
		name   string
		hbs    []heartbeat.Payload
		metric string
		ip     string
	}{
		{
			"no whitelisted version",
			[]heartbeat.Payload{{
				Version: "not_whitelisted",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 10,
			}},
			`op_heartbeat_heartbeats{chain_id="10",version="unknown"} 1`,
			"1.2.3.100",
		},
		{
			"no whitelisted chain",
			[]heartbeat.Payload{{
				Version: "v0.1.0-beta.1",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 999,
			}},
			`op_heartbeat_heartbeats{chain_id="unknown",version="v0.1.0-beta.1"} 1`,
			"1.2.3.101",
		},
		{
			"both whitelisted",
			[]heartbeat.Payload{{
				Version: "v0.1.0-beta.1",
				Meta:    "whatever",
				Moniker: "whatever",
				PeerID:  "1X2398ug",
				ChainID: 10,
			}},
			`op_heartbeat_heartbeats{chain_id="10",version="v0.1.0-beta.1"} 1`,
			"1.2.3.102",
		},
		{
			"spamming",
			[]heartbeat.Payload{
				{
					Version: "v0.1.0-goerli-rehearsal.1",
					Meta:    "whatever",
					Moniker: "alice",
					PeerID:  "1X2398ug",
					ChainID: 10,
				},
				{
					Version: "v0.1.0-goerli-rehearsal.1",
					Meta:    "whatever",
					Moniker: "bob",
					PeerID:  "1X2398ug",
					ChainID: 10,
				},
			},
			`op_heartbeat_heartbeat_same_ip_bucket{chain_id="10",version="v0.1.0-goerli-rehearsal.1",le="32"} 1`,
			"1.2.3.103",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, hb := range tt.hbs {
				data, err := json.Marshal(hb)
				require.NoError(t, err)
				req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://127.0.0.1:%d", httpPort), bytes.NewReader(data))
				require.NoError(t, err)
				req.Header.Set("X-Forwarded-For", tt.ip)
				res, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				res.Body.Close()
				require.Equal(t, res.StatusCode, 204)
			}

			metricsRes, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", metricsPort))
			require.NoError(t, err)
			defer metricsRes.Body.Close()
			require.NoError(t, err)
			metricsBody, err := io.ReadAll(metricsRes.Body)
			require.NoError(t, err)
			require.Contains(t, string(metricsBody), tt.metric)
		})
	}
}

func freePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)
	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
