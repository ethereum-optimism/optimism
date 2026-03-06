package devnet

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestRPCRecorder_RecordAndReplay(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	// Start a mock upstream that returns a fixed response
	upstream := &mockUpstream{
		responses: map[string]json.RawMessage{
			"eth_chainId": json.RawMessage(`"0x1"`),
			"eth_blockNumber": json.RawMessage(`"0x100"`),
		},
	}
	upstreamSrv := &http.Server{Handler: upstream}
	upstreamLis, err := startHTTPServer(upstreamSrv)
	require.NoError(t, err)
	t.Cleanup(func() { upstreamSrv.Close() })

	upstreamURL := "http://" + upstreamLis.Addr().String()

	// Phase 1: Record
	recorder := NewRPCRecorder(lgr, upstreamURL)
	require.NoError(t, recorder.Start())

	// Make some requests
	resp1 := rpcCall(t, recorder.Endpoint(), "eth_chainId", nil)
	require.Contains(t, string(resp1), `"0x1"`)

	resp2 := rpcCall(t, recorder.Endpoint(), "eth_blockNumber", nil)
	require.Contains(t, string(resp2), `"0x100"`)

	// Same request again (should serve from cache)
	resp3 := rpcCall(t, recorder.Endpoint(), "eth_chainId", nil)
	require.Contains(t, string(resp3), `"0x1"`)

	recording := recorder.Recording()
	require.NoError(t, recorder.Stop())

	// Verify recording contents
	require.Len(t, recording.Entries, 2) // eth_chainId and eth_blockNumber

	// Save recording
	tmpDir := t.TempDir()
	recordingPath := filepath.Join(tmpDir, "test-recording.json.gz")
	require.NoError(t, SaveRecording(recording, recordingPath))

	// Phase 2: Replay
	loaded, err := LoadRecording(recordingPath)
	require.NoError(t, err)
	require.Len(t, loaded.Entries, 2)

	replayer := NewRPCReplayer(lgr, loaded)
	require.NoError(t, replayer.Start())
	t.Cleanup(func() { replayer.Stop() })

	// Replay should return the same responses
	resp4 := rpcCall(t, replayer.Endpoint(), "eth_chainId", nil)
	require.Contains(t, string(resp4), `"0x1"`)

	resp5 := rpcCall(t, replayer.Endpoint(), "eth_blockNumber", nil)
	require.Contains(t, string(resp5), `"0x100"`)

	// Unknown method should return error in replay mode
	resp6 := rpcCall(t, replayer.Endpoint(), "eth_getBalance", json.RawMessage(`["0x1234", "latest"]`))
	require.Contains(t, string(resp6), `"error"`)
}

func TestSaveAndLoadRecording(t *testing.T) {
	recording := RPCRecording{
		Entries: map[string]*RPCRecordingEntry{
			"abc123": {
				Method:   "eth_chainId",
				Params:   json.RawMessage(`[]`),
				Response: json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`),
				Count:    1,
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json.gz")

	require.NoError(t, SaveRecording(recording, path))

	// File should exist
	_, err := os.Stat(path)
	require.NoError(t, err)

	loaded, err := LoadRecording(path)
	require.NoError(t, err)
	require.Len(t, loaded.Entries, 1)
	require.Equal(t, "eth_chainId", loaded.Entries["abc123"].Method)
}

// mockUpstream is a simple HTTP handler that returns canned JSON-RPC responses.
type mockUpstream struct {
	responses map[string]json.RawMessage
}

func (m *mockUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, ok := m.responses[req.Method]
	if !ok {
		resp := rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   json.RawMessage(`{"code":-32601,"message":"method not found"}`),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func startHTTPServer(srv *http.Server) (net.Listener, error) {
	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go srv.Serve(lis)
	return lis, nil
}

func rpcCall(t *testing.T, endpoint string, method string, params json.RawMessage) json.RawMessage {
	t.Helper()
	if params == nil {
		params = json.RawMessage(`[]`)
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return respBody
}
