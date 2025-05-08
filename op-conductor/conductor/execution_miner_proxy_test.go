package conductor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// createHTTPHandler creates a mock HTTP handler for testing, it accepts a callback which
// is invoked when the expected request is received.
func createHTTPHandler(t *testing.T, cb func(), alwaysFails bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {

				if alwaysFails {
					// TODO return a 200 but use a JSON RPC error response
					http.Error(w, "Method Not Found", http.StatusInternalServerError)
					if cb != nil {
						cb()
					}
					return
				}
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					if cb != nil {
						cb()
					}
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":true}`))
					if err != nil {
						t.Logf("Error writing response: %v", err)
					}
					return
				}
			}
		}
		http.Error(w, "Unexpected request", http.StatusBadRequest)
	}
}

func TestSetMaxDASize(t *testing.T) {
	t.Run("compliant sequencer", func(t *testing.T) {
		testSetMaxDASize(t, true)
	})
	t.Run("non-compliant sequencer", func(t *testing.T) {
		testSetMaxDASize(t, false)
	})
}

func testSetMaxDASize(t *testing.T, compliantSequencer bool) {
	ctx := context.Background()
	sequencer := httptest.NewServer(createHTTPHandler(t, nil, !compliantSequencer))
	defer sequencer.Close()

	config := mockConfig(t)
	config.ExecutionRPC = sequencer.URL
	config.NodeRPC = sequencer.URL // this won't be used but needs to be set to get the conductor to init properly
	config.RPCEnableProxy = true
	config.RPC.ListenAddr = "localhost"
	config.RPC.ListenPort = 0 // Let the system pick a random port, which we will inspect later

	conductor, err := NewOpConductor(
		ctx,
		&config,
		testlog.Logger(t, slog.LevelDebug),
		&metrics.NoopMetricsImpl{},
		"test-version",
		&clientmocks.SequencerControl{}, // not used in this test
		&consensusmocks.Consensus{},     // not used in this test
		&healthmocks.HealthMonitor{},    // not used in this test
	)

	require.NoError(t, err)

	// Start the the RPC server part of the conductor
	err = conductor.rpcServer.Start()
	require.NoError(t, err)
	defer conductor.rpcServer.Stop()

	port, err := conductor.rpcServer.Port()
	require.NoError(t, err)
	t.Log("RPC server listening on port:", port)

	url := fmt.Sprintf("http://localhost:%d", port)

	rpcClient, err := rpc.Dial(url)
	require.NoError(t, err)
	defer rpcClient.Close()

	var result bool
	err = rpcClient.CallContext(ctx, &result, "miner_setMaxDASize", "0x1", "0x2")

	if compliantSequencer {
		require.NoError(t, err)
		require.True(t, result)
		t.Log("Proxied a successful miner_setMaxDASize call")
	} else {
		require.Error(t, err)
		require.EqualError(t, err, "Simulated failure")
		t.Log("Proxied a failed miner_setMaxDASize call")
	}
}
