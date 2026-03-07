package devnet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// RPCReplayMode controls whether the replay proxy records or replays RPC calls.
type RPCReplayMode int

const (
	// RPCReplayModeRecord forwards requests to upstream and saves request/response pairs.
	RPCReplayModeRecord RPCReplayMode = iota
	// RPCReplayModeReplay serves responses from a fixture file without external calls.
	RPCReplayModeReplay
	// RPCReplayModePassthrough forwards requests to upstream without recording.
	RPCReplayModePassthrough
)

// rpcReplayEntry stores a single recorded RPC request/response pair.
type rpcReplayEntry struct {
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
	Response json.RawMessage `json:"response"`
}

// RPCReplayFixtureMetadata stores recording context alongside fixture entries.
type RPCReplayFixtureMetadata struct {
	ForkBlock uint64 `json:"fork_block,omitempty"`
}

// rpcReplayFixture is the on-disk format for recorded RPC fixtures.
type rpcReplayFixture struct {
	Metadata RPCReplayFixtureMetadata  `json:"metadata"`
	Entries  map[string]rpcReplayEntry `json:"entries"`
}

// jsonRPCRequest represents a JSON-RPC 2.0 request for parsing.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response for reconstruction.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// RPCReplayProxy is an HTTP server that either records or replays JSON-RPC exchanges.
type RPCReplayProxy struct {
	lgr         log.Logger
	upstream    string
	client      *http.Client
	mode        RPCReplayMode
	fixturePath string
	srv         *http.Server
	listenPort  int

	mu       sync.Mutex
	fixtures map[string]rpcReplayEntry
	metadata RPCReplayFixtureMetadata
}

// NewRPCReplayProxy creates a new replay proxy.
//
// In record mode, upstream must be a valid URL. In replay mode, upstream is ignored.
func NewRPCReplayProxy(lgr log.Logger, upstream string, fixturePath string, mode RPCReplayMode) *RPCReplayProxy {
	return &RPCReplayProxy{
		lgr:         lgr.New("module", "rpcreplay"),
		upstream:    upstream,
		client:      &http.Client{},
		mode:        mode,
		fixturePath: fixturePath,
		fixtures:    make(map[string]rpcReplayEntry),
	}
}

// Start loads fixtures (in replay mode) and starts the HTTP server.
func (p *RPCReplayProxy) Start() error {
	if p.mode == RPCReplayModeReplay {
		if err := p.loadFixtures(); err != nil {
			return fmt.Errorf("failed to load fixtures: %w", err)
		}
		p.lgr.Info("loaded replay fixtures", "count", len(p.fixtures), "path", p.fixturePath)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	p.listenPort = ln.Addr().(*net.TCPAddr).Port

	p.srv = &http.Server{Handler: p}

	errCh := make(chan error, 1)
	go func() {
		errCh <- p.srv.Serve(ln)
	}()

	p.lgr.Info("rpc replay proxy started", "port", p.listenPort, "mode", p.modeString())
	return nil
}

// Stop shuts down the server and, in record mode, saves fixtures to disk.
func (p *RPCReplayProxy) Stop() error {
	if p.mode == RPCReplayModeRecord {
		if err := p.saveFixtures(); err != nil {
			return fmt.Errorf("failed to save fixtures: %w", err)
		}
		p.lgr.Info("saved replay fixtures", "count", len(p.fixtures), "path", p.fixturePath)
	}
	if p.srv != nil {
		return p.srv.Close()
	}
	return nil
}

// Endpoint returns the local URL of the proxy.
func (p *RPCReplayProxy) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.listenPort)
}

// SetForkBlock records the fork block number in fixture metadata.
// Call this before Stop() so the block is persisted when fixtures are saved.
func (p *RPCReplayProxy) SetForkBlock(block uint64) {
	p.mu.Lock()
	p.metadata.ForkBlock = block
	p.mu.Unlock()
}

// ForkBlock returns the fork block from fixture metadata, if set.
func (p *RPCReplayProxy) ForkBlock() (uint64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.metadata.ForkBlock, p.metadata.ForkBlock != 0
}

func (p *RPCReplayProxy) modeString() string {
	switch p.mode {
	case RPCReplayModeRecord:
		return "record"
	case RPCReplayModePassthrough:
		return "passthrough"
	default:
		return "replay"
	}
}

func (p *RPCReplayProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Check if this is a batch request
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		p.handleBatch(w, body)
		return
	}

	p.handleSingle(w, body)
}

func (p *RPCReplayProxy) handleSingle(w http.ResponseWriter, body []byte) {
	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	key := requestKey(req.Method, req.Params)

	if p.mode == RPCReplayModeReplay {
		p.serveCached(w, req, key)
		return
	}

	// Forward to upstream (record and passthrough modes)
	respBody, err := p.forwardToUpstream(body)
	if err != nil {
		p.lgr.Error("failed to forward request", "method", req.Method, "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	if p.mode == RPCReplayModeRecord {
		var respParsed jsonRPCResponse
		if err := json.Unmarshal(respBody, &respParsed); err == nil {
			entry := rpcReplayEntry{
				Method: req.Method,
				Params: req.Params,
			}
			if respParsed.Error != nil {
				entry.Response = mustMarshal(map[string]json.RawMessage{"error": respParsed.Error})
			} else {
				entry.Response = mustMarshal(map[string]json.RawMessage{"result": respParsed.Result})
			}
			p.mu.Lock()
			p.fixtures[key] = entry
			p.mu.Unlock()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func (p *RPCReplayProxy) handleBatch(w http.ResponseWriter, body []byte) {
	var reqs []jsonRPCRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		http.Error(w, "invalid batch JSON-RPC request", http.StatusBadRequest)
		return
	}

	if p.mode == RPCReplayModeReplay {
		responses := make([]json.RawMessage, len(reqs))
		for i, req := range reqs {
			key := requestKey(req.Method, req.Params)
			p.mu.Lock()
			entry, ok := p.fixtures[key]
			p.mu.Unlock()
			if !ok {
				p.lgr.Error("no fixture found for batch request", "method", req.Method, "key", key)
				http.Error(w, fmt.Sprintf("no fixture for method %s", req.Method), http.StatusNotFound)
				return
			}
			responses[i] = reconstructResponse(req.ID, entry.Response)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc, _ := json.Marshal(responses)
		_, _ = w.Write(enc)
		return
	}

	// Record/passthrough: forward batch to upstream
	respBody, err := p.forwardToUpstream(body)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	if p.mode == RPCReplayModeRecord {
		var resps []jsonRPCResponse
		if err := json.Unmarshal(respBody, &resps); err == nil {
			for i, req := range reqs {
				if i >= len(resps) {
					break
				}
				key := requestKey(req.Method, req.Params)
				entry := rpcReplayEntry{
					Method: req.Method,
					Params: req.Params,
				}
				if resps[i].Error != nil {
					entry.Response = mustMarshal(map[string]json.RawMessage{"error": resps[i].Error})
				} else {
					entry.Response = mustMarshal(map[string]json.RawMessage{"result": resps[i].Result})
				}
				p.mu.Lock()
				p.fixtures[key] = entry
				p.mu.Unlock()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func (p *RPCReplayProxy) serveCached(w http.ResponseWriter, req jsonRPCRequest, key string) {
	p.mu.Lock()
	entry, ok := p.fixtures[key]
	p.mu.Unlock()

	if !ok {
		p.lgr.Error("no fixture found", "method", req.Method, "key", key, "params", string(req.Params))
		http.Error(w, fmt.Sprintf("no fixture for method %s (key %s)", req.Method, key), http.StatusNotFound)
		return
	}

	resp := reconstructResponse(req.ID, entry.Response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func (p *RPCReplayProxy) forwardToUpstream(body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, p.upstream, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (p *RPCReplayProxy) loadFixtures() error {
	data, err := os.ReadFile(p.fixturePath)
	if err != nil {
		return err
	}

	var fixture rpcReplayFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return err
	}

	p.mu.Lock()
	p.fixtures = fixture.Entries
	p.metadata = fixture.Metadata
	p.mu.Unlock()
	return nil
}

func (p *RPCReplayProxy) saveFixtures() error {
	p.mu.Lock()
	fixtures := make(map[string]rpcReplayEntry, len(p.fixtures))
	for k, v := range p.fixtures {
		fixtures[k] = v
	}
	p.mu.Unlock()

	fixture := rpcReplayFixture{Metadata: p.metadata, Entries: fixtures}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p.fixturePath, data, 0o644)
}

// requestKey generates a deterministic cache key from method + params.
func requestKey(method string, params json.RawMessage) string {
	// Normalize params: parse and re-marshal to get canonical JSON
	var normalized interface{}
	if err := json.Unmarshal(params, &normalized); err != nil {
		// Fallback: use raw params
		normalized = string(params)
	}
	canonical, _ := json.Marshal(normalized)

	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(":"))
	h.Write(canonical)
	return method + ":" + hex.EncodeToString(h.Sum(nil))[:16]
}

// reconstructResponse builds a full JSON-RPC response from a cached entry.
func reconstructResponse(id json.RawMessage, cached json.RawMessage) json.RawMessage {
	var parts map[string]json.RawMessage
	if err := json.Unmarshal(cached, &parts); err != nil {
		return cached
	}

	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  parts["result"],
		Error:   parts["error"],
	}
	data, _ := json.Marshal(resp)
	return data
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// ReadFixtureMetadata reads just the metadata from a fixture file.
// Returns zero-valued metadata if the file doesn't exist or has no metadata.
func ReadFixtureMetadata(fixturePath string) (RPCReplayFixtureMetadata, error) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return RPCReplayFixtureMetadata{}, err
	}
	var fixture rpcReplayFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return RPCReplayFixtureMetadata{}, err
	}
	return fixture.Metadata, nil
}

// FixtureForkBlock returns the fork block from a fixture file's metadata.
// If the file doesn't exist or has no fork_block metadata, returns fallback.
func FixtureForkBlock(fixturePath string, fallback uint64) uint64 {
	meta, err := ReadFixtureMetadata(fixturePath)
	if err != nil || meta.ForkBlock == 0 {
		return fallback
	}
	return meta.ForkBlock
}

// FixtureKeys returns sorted keys from a fixture file, useful for debugging.
func FixtureKeys(fixturePath string) ([]string, error) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, err
	}
	var fixture rpcReplayFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(fixture.Entries))
	for k := range fixture.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}
