package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// HTTPAdapter is the production [Adapter]: it reads a remote chain's finalized
// initiating messages from an HTTP endpoint. Point it at any server that speaks the
// remote-node protocol below — that server is the real plugin boundary, so a remote
// chain can be backed by any node software, in any language, as long as it responds
// correctly.
//
// Protocol:
//
//	GET {baseURL}/finalized?after={N}
//	200 application/json: {"blockTime": <secs>, "block": <FinalizedBlock>|null}
//	  - block is the finalized block with Number == N+1, or null if no block after N
//	    is finalized yet (the client then waits and retries).
//	  - N == 0 is the anchor request (the supernode has no history for this chain
//	    yet): the server answers with the block it wants history to start at, at any
//	    height >= 1, and must keep answering with that same block until the supernode
//	    has ingested past it. See Adapter.NextFinalized.
//	  - blockTime is the remote chain's block time in seconds (always present); it is
//	    cached and surfaced via BlockTime for the interop activation invariant.
type HTTPAdapter struct {
	chainID   eth.ChainID
	baseURL   string
	client    *http.Client
	blockTime atomic.Uint64
}

// finalizedResponse is the wire envelope returned by GET /finalized.
type finalizedResponse struct {
	BlockTime uint64          `json:"blockTime"`
	Block     *FinalizedBlock `json:"block"`
}

var _ Adapter = (*HTTPAdapter)(nil)

// NewHTTPAdapter builds an HTTPAdapter for the given chain and base URL. A nil client
// uses http.DefaultClient.
func NewHTTPAdapter(chainID eth.ChainID, baseURL string, client *http.Client) *HTTPAdapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPAdapter{
		chainID: chainID,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (a *HTTPAdapter) ChainID() eth.ChainID { return a.chainID }

// BlockTime returns the remote chain's block time as last reported by the endpoint, or
// 0 before the first successful response. Since a message can only be referenced after
// its block has been ingested, a non-zero value is always available by validation time.
func (a *HTTPAdapter) BlockTime() uint64 { return a.blockTime.Load() }

func (a *HTTPAdapter) NextFinalized(ctx context.Context, after uint64) (*FinalizedBlock, bool, error) {
	url := fmt.Sprintf("%s/finalized?after=%d", a.baseURL, after)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
	var out finalizedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false, fmt.Errorf("decode response from %s: %w", url, err)
	}
	if out.BlockTime > 0 {
		a.blockTime.Store(out.BlockTime)
	}
	if out.Block == nil {
		return nil, false, nil
	}
	return out.Block, true, nil
}

func (a *HTTPAdapter) Close() error { return nil }
