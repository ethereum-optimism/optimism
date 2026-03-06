package devnet

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// RPCRecording stores recorded RPC request/response pairs indexed by a
// deterministic key derived from the method and params.
type RPCRecording struct {
	Entries map[string]*RPCRecordingEntry `json:"entries"`
}

// RPCRecordingEntry is a single recorded RPC exchange.
type RPCRecordingEntry struct {
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
	Response json.RawMessage `json:"response"`
	Count    int             `json:"count"`
}

// rpcRequest is the JSON-RPC request structure.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// rpcResponse is the JSON-RPC response structure.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func recordingKey(method string, params json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(":"))
	h.Write(normalizeJSON(params))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func normalizeJSON(data json.RawMessage) []byte {
	if len(data) == 0 {
		return []byte("null")
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return data
	}
	out, err := json.Marshal(v)
	if err != nil {
		return data
	}
	return out
}

// RPCRecorder is an HTTP proxy that records or replays JSON-RPC exchanges.
type RPCRecorder struct {
	lgr       log.Logger
	upstream  string
	client    *http.Client
	recording RPCRecording
	mu        sync.Mutex
	srv       *http.Server
	lis       net.Listener
	replayOnly bool
}

// NewRPCRecorder creates a new recorder in record mode (with upstream).
func NewRPCRecorder(lgr log.Logger, upstream string) *RPCRecorder {
	return &RPCRecorder{
		lgr:       lgr.New("module", "rpcrecorder"),
		upstream:  upstream,
		client:    &http.Client{Timeout: 30 * time.Second},
		recording: RPCRecording{Entries: make(map[string]*RPCRecordingEntry)},
	}
}

// NewRPCReplayer creates a recorder in replay-only mode (no upstream).
func NewRPCReplayer(lgr log.Logger, recording RPCRecording) *RPCRecorder {
	return &RPCRecorder{
		lgr:        lgr.New("module", "rpcreplayer"),
		recording:  recording,
		replayOnly: true,
	}
}

func (r *RPCRecorder) Start() error {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	r.lis = lis

	r.srv = &http.Server{Handler: r}

	errCh := make(chan error, 1)
	go func() {
		if err := r.srv.Serve(lis); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	timer := time.NewTimer(100 * time.Millisecond)
	select {
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case <-timer.C:
		r.lgr.Info("RPC recorder started", "addr", lis.Addr().String(), "replay_only", r.replayOnly)
		return nil
	}
}

func (r *RPCRecorder) Stop() error {
	if r.srv == nil {
		return nil
	}
	return r.srv.Shutdown(context.Background())
}

func (r *RPCRecorder) Endpoint() string {
	return fmt.Sprintf("http://%s", r.lis.Addr().String())
}

func (r *RPCRecorder) Recording() RPCRecording {
	r.mu.Lock()
	defer r.mu.Unlock()

	copied := RPCRecording{Entries: make(map[string]*RPCRecordingEntry, len(r.recording.Entries))}
	for k, v := range r.recording.Entries {
		entry := *v
		copied.Entries[k] = &entry
	}
	return copied
}

func (r *RPCRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Handle batch requests
	if len(body) > 0 && body[0] == '[' {
		r.handleBatch(w, body)
		return
	}

	r.handleSingle(w, body)
}

func (r *RPCRecorder) handleSingle(w http.ResponseWriter, body []byte) {
	var rpcReq rpcRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	key := recordingKey(rpcReq.Method, rpcReq.Params)

	// Try cache first
	r.mu.Lock()
	entry, cached := r.recording.Entries[key]
	r.mu.Unlock()

	if cached {
		r.lgr.Trace("cache hit", "method", rpcReq.Method, "key", key)
		// Rewrite the response ID to match the request
		resp := r.rewriteID(entry.Response, rpcReq.ID)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
		return
	}

	if r.replayOnly {
		r.lgr.Error("cache miss in replay mode", "method", rpcReq.Method, "key", key, "params", string(rpcReq.Params))
		errResp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"no recording for %s"}}`, rpcReq.ID, rpcReq.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(errResp))
		return
	}

	// Forward to upstream
	upstreamResp, err := r.forwardToUpstream(body)
	if err != nil {
		r.lgr.Error("upstream request failed", "method", rpcReq.Method, "err", err)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}

	// Record the exchange
	r.mu.Lock()
	r.recording.Entries[key] = &RPCRecordingEntry{
		Method:   rpcReq.Method,
		Params:   rpcReq.Params,
		Response: upstreamResp,
		Count:    1,
	}
	r.mu.Unlock()

	r.lgr.Trace("recorded", "method", rpcReq.Method, "key", key)

	// Rewrite ID and serve
	resp := r.rewriteID(upstreamResp, rpcReq.ID)
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (r *RPCRecorder) handleBatch(w http.ResponseWriter, body []byte) {
	var reqs []rpcRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		http.Error(w, "invalid batch request", http.StatusBadRequest)
		return
	}

	// For batch requests in replay mode, try to serve all from cache
	if r.replayOnly {
		var resps []json.RawMessage
		for _, rpcReq := range reqs {
			key := recordingKey(rpcReq.Method, rpcReq.Params)
			r.mu.Lock()
			entry, cached := r.recording.Entries[key]
			r.mu.Unlock()

			if cached {
				resps = append(resps, r.rewriteID(entry.Response, rpcReq.ID))
			} else {
				errResp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"no recording for %s"}}`, rpcReq.ID, rpcReq.Method)
				resps = append(resps, json.RawMessage(errResp))
			}
		}

		out, _ := json.Marshal(resps)
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	// In record mode, forward the whole batch
	upstreamResp, err := r.forwardToUpstream(body)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}

	// Parse response to record individual entries
	var resps []json.RawMessage
	if err := json.Unmarshal(upstreamResp, &resps); err == nil {
		for i, rpcReq := range reqs {
			if i < len(resps) {
				key := recordingKey(rpcReq.Method, rpcReq.Params)
				r.mu.Lock()
				r.recording.Entries[key] = &RPCRecordingEntry{
					Method:   rpcReq.Method,
					Params:   rpcReq.Params,
					Response: resps[i],
					Count:    1,
				}
				r.mu.Unlock()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(upstreamResp)
}

func (r *RPCRecorder) forwardToUpstream(body []byte) (json.RawMessage, error) {
	resp, err := r.client.Post(r.upstream, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(respBody), nil
}

func (r *RPCRecorder) rewriteID(response json.RawMessage, newID json.RawMessage) json.RawMessage {
	var parsed rpcResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return response
	}
	parsed.ID = newID
	out, err := json.Marshal(parsed)
	if err != nil {
		return response
	}
	return out
}

// SaveRecording writes the recording to a gzip-compressed JSON file.
func SaveRecording(recording RPCRecording, path string) error {
	// Sort entries by key for deterministic output
	keys := make([]string, 0, len(recording.Entries))
	for k := range recording.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	enc := json.NewEncoder(gw)
	enc.SetIndent("", "  ")
	if err := enc.Encode(recording); err != nil {
		return fmt.Errorf("failed to encode recording: %w", err)
	}

	return nil
}

// LoadRecording reads a gzip-compressed JSON recording file.
func LoadRecording(path string) (RPCRecording, error) {
	f, err := os.Open(path)
	if err != nil {
		return RPCRecording{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return RPCRecording{}, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	var recording RPCRecording
	if err := json.NewDecoder(gr).Decode(&recording); err != nil {
		return RPCRecording{}, fmt.Errorf("failed to decode recording: %w", err)
	}

	return recording, nil
}
