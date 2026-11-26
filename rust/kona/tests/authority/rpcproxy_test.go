package authority_test

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
)

// rpcProxy serves a kona-node a JSON-RPC endpoint over HTTP, and can cut the node off from it, or,
// for an L1 endpoint, freeze the node's view of L1 at a block: while frozen, the latest block is
// that block and later blocks do not exist, as for a node whose view of L1 lags behind the chain.
type rpcProxy struct {
	upstream string
	// frozen is the block number the view is frozen at, or -1.
	frozen atomic.Int64
	cut    atomic.Bool
	srv    *httptest.Server
}

func newRPCProxy(t helpers.Testing, upstream string) *rpcProxy {
	v := &rpcProxy{upstream: upstream}
	v.frozen.Store(-1)
	v.srv = httptest.NewServer(v)
	t.Cleanup(v.srv.Close)
	return v
}

func (v *rpcProxy) url() string { return v.srv.URL }

// freeze stops the node's view of L1 at block number at.
func (v *rpcProxy) freeze(at uint64) { v.frozen.Store(int64(at)) }

// unfreeze shows the node the whole L1 chain again.
func (v *rpcProxy) unfreeze() { v.frozen.Store(-1) }

// disconnect fails every request of the node, as for a lost connection, until reconnect.
func (v *rpcProxy) disconnect() { v.cut.Store(true) }

func (v *rpcProxy) reconnect() { v.cut.Store(false) }

func (v *rpcProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if v.cut.Load() {
		http.Error(w, "disconnected", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if at := v.frozen.Load(); at >= 0 {
		body = hideAfter(body, uint64(at))
	}
	resp, err := http.Post(v.upstream, "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

type rpcRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id,omitempty"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params,omitempty"`
}

// hideAfter rewrites the JSON-RPC requests in body that read a block by number, so that the
// latest block is block at and no later block exists. Other requests pass unchanged.
func hideAfter(body []byte, at uint64) []byte {
	trimmed := bytes.TrimSpace(body)
	batch := bytes.HasPrefix(trimmed, []byte("["))
	var reqs []rpcRequest
	if batch {
		if json.Unmarshal(trimmed, &reqs) != nil {
			return body
		}
	} else {
		var req rpcRequest
		if json.Unmarshal(trimmed, &req) != nil {
			return body
		}
		reqs = []rpcRequest{req}
	}
	for i := range reqs {
		if reqs[i].Method != "eth_getBlockByNumber" || len(reqs[i].Params) == 0 {
			continue
		}
		var tag string
		if json.Unmarshal(reqs[i].Params[0], &tag) != nil {
			continue
		}
		switch tag {
		case "latest", "pending":
			reqs[i].Params[0] = mustJSON(hexutil.EncodeUint64(at))
		case "safe", "finalized", "earliest":
		default:
			if n, err := hexutil.DecodeUint64(tag); err == nil && n > at {
				// No block has this number, so the L1 node answers none.
				reqs[i].Params[0] = mustJSON(hexutil.EncodeUint64(math.MaxInt64))
			}
		}
	}
	var out []byte
	var err error
	if batch {
		out, err = json.Marshal(reqs)
	} else {
		out, err = json.Marshal(reqs[0])
	}
	if err != nil {
		return body
	}
	return out
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
