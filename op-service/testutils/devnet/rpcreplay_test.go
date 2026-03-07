package devnet

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func newFakeUpstream(t *testing.T) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		trimmed := bytes.TrimSpace(body)

		w.Header().Set("Content-Type", "application/json")

		// Handle batch
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var reqs []jsonRPCRequest
			_ = json.Unmarshal(body, &reqs)
			var resps []map[string]interface{}
			for _, req := range reqs {
				resps = append(resps, map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(req.ID),
					"result":  json.RawMessage(fakeResult(req.Method)),
				})
			}
			_ = json.NewEncoder(w).Encode(resps)
			return
		}

		var req jsonRPCRequest
		_ = json.Unmarshal(body, &req)
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(req.ID),
			"result":  json.RawMessage(fakeResult(req.Method)),
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	url := "http://" + ln.Addr().String()
	return url, func() { srv.Close() }
}

func fakeResult(method string) string {
	switch method {
	case "eth_chainId":
		return `"0xaa36a7"`
	case "eth_blockNumber":
		return `"0x7a1200"`
	default:
		return `null`
	}
}

func TestRPCReplayProxy_RecordAndReplay(t *testing.T) {
	lgr := log.NewLogger(log.DiscardHandler())
	upstreamURL, cleanup := newFakeUpstream(t)
	defer cleanup()

	fixturePath := filepath.Join(t.TempDir(), "fixtures.json")

	// Phase 1: Record
	recorder := NewRPCReplayProxy(lgr, upstreamURL, fixturePath, RPCReplayModeRecord)
	require.NoError(t, recorder.Start())

	chainIDResp := doRPCCall(t, recorder.Endpoint(), "eth_chainId", `[]`)
	require.Contains(t, string(chainIDResp), "0xaa36a7")

	blockResp := doRPCCall(t, recorder.Endpoint(), "eth_blockNumber", `[]`)
	require.Contains(t, string(blockResp), "0x7a1200")

	require.NoError(t, recorder.Stop())
	require.FileExists(t, fixturePath)

	// Phase 2: Replay (no upstream)
	replayer := NewRPCReplayProxy(lgr, "", fixturePath, RPCReplayModeReplay)
	require.NoError(t, replayer.Start())

	chainIDResp2 := doRPCCall(t, replayer.Endpoint(), "eth_chainId", `[]`)
	require.Contains(t, string(chainIDResp2), "0xaa36a7")

	blockResp2 := doRPCCall(t, replayer.Endpoint(), "eth_blockNumber", `[]`)
	require.Contains(t, string(blockResp2), "0x7a1200")

	// Unknown method should 404
	resp, err := http.Post(replayer.Endpoint(), "application/json",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"eth_unknownMethod","params":[]}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	require.NoError(t, replayer.Stop())
}

func TestRPCReplayProxy_BatchRequests(t *testing.T) {
	lgr := log.NewLogger(log.DiscardHandler())
	upstreamURL, cleanup := newFakeUpstream(t)
	defer cleanup()

	fixturePath := filepath.Join(t.TempDir(), "batch_fixtures.json")

	// Record
	recorder := NewRPCReplayProxy(lgr, upstreamURL, fixturePath, RPCReplayModeRecord)
	require.NoError(t, recorder.Start())

	batchReq := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]},{"jsonrpc":"2.0","id":2,"method":"eth_blockNumber","params":[]}]`
	resp, err := http.Post(recorder.Endpoint(), "application/json", bytes.NewReader([]byte(batchReq)))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Contains(t, string(body), "0xaa36a7")

	require.NoError(t, recorder.Stop())

	// Replay
	replayer := NewRPCReplayProxy(lgr, "", fixturePath, RPCReplayModeReplay)
	require.NoError(t, replayer.Start())

	resp2, err := http.Post(replayer.Endpoint(), "application/json", bytes.NewReader([]byte(batchReq)))
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	require.Contains(t, string(body2), "0xaa36a7")
	require.Contains(t, string(body2), "0x7a1200")

	require.NoError(t, replayer.Stop())
}

func TestRequestKey_Deterministic(t *testing.T) {
	key1 := requestKey("eth_getBalance", json.RawMessage(`["0xabc", "latest"]`))
	key2 := requestKey("eth_getBalance", json.RawMessage(`["0xabc", "latest"]`))
	key3 := requestKey("eth_getBalance", json.RawMessage(`["0xdef", "latest"]`))

	require.Equal(t, key1, key2)
	require.NotEqual(t, key1, key3)
}

func TestFixtureKeys(t *testing.T) {
	fixturePath := filepath.Join(t.TempDir(), "test.json")
	fixture := rpcReplayFixture{
		Entries: map[string]rpcReplayEntry{
			"eth_chainId:abc123":     {Method: "eth_chainId"},
			"eth_blockNumber:def456": {Method: "eth_blockNumber"},
		},
	}
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fixturePath, data, 0o644))

	keys, err := FixtureKeys(fixturePath)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.Equal(t, "eth_blockNumber:def456", keys[0])
	require.Equal(t, "eth_chainId:abc123", keys[1])
}

func TestRPCReplayOrRecord_PrefersFixtures(t *testing.T) {
	lgr := log.NewLogger(log.DiscardHandler())

	// Create a fixture file with a known response
	fixturePath := filepath.Join(t.TempDir(), "test.json")
	fixture := rpcReplayFixture{
		Entries: map[string]rpcReplayEntry{
			requestKey("eth_chainId", json.RawMessage(`[]`)): {
				Method:   "eth_chainId",
				Params:   json.RawMessage(`[]`),
				Response: json.RawMessage(`{"result":"0xaa36a7"}`),
			},
		},
	}
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fixturePath, data, 0o644))

	// Even with SEPOLIA_RPC_URL set, fixtures should be used (replay mode)
	t.Setenv("SEPOLIA_RPC_URL", "http://127.0.0.1:1") // Invalid URL — proves we don't hit it
	t.Setenv("RPC_REPLAY_RECORD", "")

	proxy, cleanup, err := RPCReplayOrRecord(lgr, fixturePath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })

	resp := doRPCCall(t, proxy.Endpoint(), "eth_chainId", `[]`)
	require.Contains(t, string(resp), "0xaa36a7")
}

func TestRPCReplayProxy_MetadataRoundTrip(t *testing.T) {
	lgr := log.NewLogger(log.DiscardHandler())
	upstreamURL, cleanup := newFakeUpstream(t)
	defer cleanup()

	fixturePath := filepath.Join(t.TempDir(), "meta_fixtures.json")

	// Record with fork block metadata
	recorder := NewRPCReplayProxy(lgr, upstreamURL, fixturePath, RPCReplayModeRecord)
	recorder.SetForkBlock(12345)
	require.NoError(t, recorder.Start())

	doRPCCall(t, recorder.Endpoint(), "eth_chainId", `[]`)
	require.NoError(t, recorder.Stop())

	// Verify metadata was saved
	meta, err := ReadFixtureMetadata(fixturePath)
	require.NoError(t, err)
	require.Equal(t, uint64(12345), meta.ForkBlock)

	// Replay should load metadata
	replayer := NewRPCReplayProxy(lgr, "", fixturePath, RPCReplayModeReplay)
	require.NoError(t, replayer.Start())

	block, ok := replayer.ForkBlock()
	require.True(t, ok)
	require.Equal(t, uint64(12345), block)

	require.NoError(t, replayer.Stop())
}

func TestFixtureForkBlock_Fallback(t *testing.T) {
	// Missing file returns fallback
	block := FixtureForkBlock(filepath.Join(t.TempDir(), "nonexistent.json"), 99999)
	require.Equal(t, uint64(99999), block)

	// Fixture without metadata returns fallback
	fixturePath := filepath.Join(t.TempDir(), "no_meta.json")
	fixture := rpcReplayFixture{
		Entries: map[string]rpcReplayEntry{
			"eth_chainId:abc": {Method: "eth_chainId"},
		},
	}
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fixturePath, data, 0o644))

	block = FixtureForkBlock(fixturePath, 99999)
	require.Equal(t, uint64(99999), block)

	// Fixture with metadata returns the stored block
	fixtureWithMeta := rpcReplayFixture{
		Metadata: RPCReplayFixtureMetadata{ForkBlock: 42},
		Entries:  map[string]rpcReplayEntry{},
	}
	data, err = json.Marshal(fixtureWithMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fixturePath, data, 0o644))

	block = FixtureForkBlock(fixturePath, 99999)
	require.Equal(t, uint64(42), block)
}

func TestRPCReplayOrRecord_NoFixturesNoRPC(t *testing.T) {
	lgr := log.NewLogger(log.DiscardHandler())

	fixturePath := filepath.Join(t.TempDir(), "nonexistent.json")
	t.Setenv("SEPOLIA_RPC_URL", "")
	t.Setenv("RPC_REPLAY_RECORD", "")

	_, _, err := RPCReplayOrRecord(lgr, fixturePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no fixture file")
}

func doRPCCall(t *testing.T, endpoint, method, params string) []byte {
	t.Helper()
	reqBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return body
}
