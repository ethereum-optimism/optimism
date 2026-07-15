package proposer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type jsonRPCRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
}

func TestProposalSourceUsesRootFormat(t *testing.T) {
	testCases := []struct {
		name           string
		gameType       uint32
		useRollupRPC   bool
		expectedMethod string
	}{
		{name: "CannonKonaRollup", gameType: 8, useRollupRPC: true, expectedMethod: "optimism_outputAtBlock"},
		{name: "SuperPermissionedRollup", gameType: 5, useRollupRPC: true, expectedMethod: "superroot_atTimestamp"},
		{name: "SuperCannonKonaRollup", gameType: 9, useRollupRPC: true, expectedMethod: "superroot_atTimestamp"},
		{name: "SuperCannonKonaSuperNode", gameType: 9, expectedMethod: "superroot_atTimestamp"},
		{name: "UnknownRollupFallback", gameType: 492743, useRollupRPC: true, expectedMethod: "optimism_outputAtBlock"},
		{name: "UnknownSuperNodeFallback", gameType: 492743, expectedMethod: "superroot_atTimestamp"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server, methods := recordingRPCServer(t)
			cfg := validConfig()
			cfg.L1EthRpc = server.URL
			cfg.DisputeGameType = testCase.gameType
			if testCase.useRollupRPC {
				cfg.RollupRpc = server.URL
				cfg.SuperNodeRpcs = nil
			} else {
				cfg.RollupRpc = ""
				cfg.SuperNodeRpcs = []string{server.URL}
			}

			service := ProposerService{
				Log:     testlog.Logger(t, log.LevelInfo),
				Metrics: metrics.NoopMetrics,
			}
			require.NoError(t, service.initRPCClients(context.Background(), cfg))
			t.Cleanup(func() {
				service.L1Client.Close()
				service.ProposalSource.Close()
			})

			_, err := service.ProposalSource.ProposalAtSequenceNum(context.Background(), 123)
			require.Error(t, err)

			select {
			case method := <-methods:
				require.Equal(t, testCase.expectedMethod, method)
			case <-time.After(time.Second):
				t.Fatal("proposal source did not make an RPC request")
			}
		})
	}
}

func recordingRPCServer(t *testing.T) (*httptest.Server, <-chan string) {
	t.Helper()
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		methods <- request.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"error": map[string]any{
				"code":    -32000,
				"message": "recorded",
			},
		})
	}))
	t.Cleanup(server.Close)
	return server, methods
}
