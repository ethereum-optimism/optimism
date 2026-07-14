package rustengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// forkFixture is a recorded set of JSON-RPC responses for the read-only fork surface (the 5 methods
// forking.RPCSource + revm AlloyDB use), pinned to one immutable L1 block. It lets the A/B fork
// parity gate run fully hermetic — no network, no secret, no anvil — in the required go-tests-short
// CI job. Recorded once against a public Sepolia archive (record mode below); committed to testdata.
type forkFixture struct {
	BlockNumber uint64                     `json:"blockNumber"`
	BlockHash   common.Hash                `json:"blockHash"`
	Entries     map[string]json.RawMessage `json:"entries"`
}

// fixtureServer replays a forkFixture over HTTP JSON-RPC, serving BOTH engines (the Go host dials it
// via rpc.Dial, the Rust engine via script_createSelectFork) so their fork base state is identical
// by construction. In record mode it forwards cache misses to an upstream archive and accumulates
// the union of both engines' requests, then dumps the fixture.
type fixtureServer struct {
	mu       sync.Mutex
	fx       *forkFixture
	record   bool
	upstream string
	client   *http.Client
	// missed records replay-mode cache misses so the test fails loudly (never silently 0-fills).
	missed []string
}

type jsonRPCReq struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
}

type jsonRPCResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// canonicalNum normalizes a block-number tag ("0xabae10", "latest", ...) to a stable key fragment.
func canonicalNum(tag string) string {
	if strings.HasPrefix(tag, "0x") {
		if n, ok := new(big.Int).SetString(tag[2:], 16); ok {
			return "0x" + n.Text(16)
		}
	}
	return tag
}

// fixtureKey collapses a request to a block-encoding-independent key, so the Go host (bare block-hash
// string) and the Rust engine (EIP-1898 {"blockHash":...} object, minimal-quantity storage slots)
// hit the same recorded entry. All account reads are pinned to the one fixture block, so the block
// param is intentionally dropped from the key.
func fixtureKey(method string, params []json.RawMessage) string {
	str := func(raw json.RawMessage) string {
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	}
	switch method {
	case "eth_getBlockByNumber":
		return method + ":" + canonicalNum(str(params[0]))
	case "eth_getBlockByHash":
		return method + ":" + strings.ToLower(str(params[0]))
	case "eth_getBalance", "eth_getTransactionCount", "eth_getCode":
		return method + ":" + strings.ToLower(common.HexToAddress(str(params[0])).Hex())
	case "eth_getStorageAt":
		return method + ":" + strings.ToLower(common.HexToAddress(str(params[0])).Hex()) +
			":" + common.HexToHash(str(params[1])).Hex()
	default:
		var b bytes.Buffer
		for _, p := range params {
			b.Write(p)
			b.WriteByte(',')
		}
		return method + ":" + b.String()
	}
}

func (s *fixtureServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trimmed := bytes.TrimSpace(body)
	w.Header().Set("Content-Type", "application/json")
	if len(trimmed) > 0 && trimmed[0] == '[' { // batch
		var reqs []jsonRPCReq
		if err := json.Unmarshal(trimmed, &reqs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resps := make([]jsonRPCResp, len(reqs))
		for i, req := range reqs {
			resps[i] = s.dispatch(req)
		}
		_ = json.NewEncoder(w).Encode(resps)
		return
	}
	var req jsonRPCReq
	if err := json.Unmarshal(trimmed, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(s.dispatch(req))
}

func (s *fixtureServer) dispatch(req jsonRPCReq) jsonRPCResp {
	key := fixtureKey(req.Method, req.Params)
	s.mu.Lock()
	res, ok := s.fx.Entries[key]
	s.mu.Unlock()
	if ok {
		return jsonRPCResp{JSONRPC: "2.0", ID: req.ID, Result: res}
	}
	if !s.record {
		s.mu.Lock()
		s.missed = append(s.missed, key)
		s.mu.Unlock()
		return jsonRPCResp{JSONRPC: "2.0", ID: req.ID,
			Error: &jsonRPCError{Code: -32000, Message: "fork fixture miss: " + key}}
	}
	// Record mode: forward the original request (original block encoding) to the upstream archive.
	res, rpcErr := s.fetchUpstream(req)
	if rpcErr != nil {
		return jsonRPCResp{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
	}
	s.mu.Lock()
	s.fx.Entries[key] = res
	s.mu.Unlock()
	return jsonRPCResp{JSONRPC: "2.0", ID: req.ID, Result: res}
}

func (s *fixtureServer) fetchUpstream(req jsonRPCReq) (json.RawMessage, *jsonRPCError) {
	fwd, _ := json.Marshal(jsonRPCReq{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: req.Method, Params: req.Params})
	resp, err := s.client.Post(s.upstream, "application/json", bytes.NewReader(fwd))
	if err != nil {
		return nil, &jsonRPCError{Code: -32000, Message: "upstream: " + err.Error()}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &jsonRPCError{Code: -32000, Message: "upstream read: " + err.Error()}
	}
	var out jsonRPCResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &jsonRPCError{Code: -32000, Message: "upstream decode: " + err.Error()}
	}
	if out.Error != nil {
		return nil, out.Error
	}
	return out.Result, nil
}

const forkFixturePath = "testdata/fork/setdisputegameimpl-sepolia.json"

// startForkFixtureServer loads the committed fixture (replay mode) or, when RECORD_FORK_FIXTURE=1,
// starts an empty recorder forwarding to SEPOLIA_RPC_URL (falling back to the public ethpandaops
// archive). Returns the running httptest server and a dump func (record mode writes the fixture).
func startForkFixtureServer(t *testing.T, wantBlock uint64) (*httptest.Server, func()) {
	t.Helper()
	fs := &fixtureServer{client: &http.Client{}}

	if os.Getenv("RECORD_FORK_FIXTURE") == "1" {
		up := os.Getenv("SEPOLIA_RPC_URL")
		if up == "" {
			up = "https://rpc.sepolia.ethpandaops.io"
		}
		fs.record = true
		fs.upstream = up
		fs.fx = &forkFixture{BlockNumber: wantBlock, Entries: map[string]json.RawMessage{}}
	} else {
		raw, err := os.ReadFile(forkFixturePath)
		require.NoError(t, err, "committed fork fixture missing; regenerate with RECORD_FORK_FIXTURE=1")
		var fx forkFixture
		require.NoError(t, json.Unmarshal(raw, &fx))
		require.Equal(t, wantBlock, fx.BlockNumber, "fixture pinned to a different block")
		fs.fx = &fx
	}

	srv := httptest.NewServer(http.HandlerFunc(fs.handle))
	dump := func() {
		if !fs.record {
			// Replay mode: fail loudly on any cache miss (never a silent 0-fill divergence).
			require.Empty(t, fs.missed, "fork fixture cache misses (regenerate the fixture): %v", dedup(fs.missed))
			return
		}
		// Resolve the recorded block hash for the fixture metadata.
		if res, ok := fs.fx.Entries["eth_getBlockByNumber:"+canonicalNum(fmt.Sprintf("0x%x", wantBlock))]; ok {
			var hdr struct {
				Hash common.Hash `json:"hash"`
			}
			require.NoError(t, json.Unmarshal(res, &hdr))
			fs.fx.BlockHash = hdr.Hash
		}
		writeFixture(t, fs.fx)
	}
	return srv, dump
}

func writeFixture(t *testing.T, fx *forkFixture) {
	t.Helper()
	require.NoError(t, os.MkdirAll("testdata/fork", 0o755))
	// Stable, sorted, indented encoding so the committed file diffs cleanly.
	keys := make([]string, 0, len(fx.Entries))
	for k := range fx.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"blockNumber\": %d,\n", fx.BlockNumber))
	b.WriteString(fmt.Sprintf("  \"blockHash\": %q,\n", fx.BlockHash.Hex()))
	b.WriteString("  \"entries\": {\n")
	for i, k := range keys {
		comma := ","
		if i == len(keys)-1 {
			comma = ""
		}
		b.WriteString(fmt.Sprintf("    %q: %s%s\n", k, string(fx.Entries[k]), comma))
	}
	b.WriteString("  }\n}\n")
	require.NoError(t, os.WriteFile(forkFixturePath, b.Bytes(), 0o644))
	t.Logf("recorded fork fixture: %d entries -> %s", len(fx.Entries), forkFixturePath)
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
