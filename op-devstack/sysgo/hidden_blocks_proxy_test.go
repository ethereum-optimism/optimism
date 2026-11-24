package sysgo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestHiddenBlocksRefusesRequestsNamingAHiddenBlock relays requests to an EL
// that answers every one, and checks which the proxy answers itself instead.
func TestHiddenBlocksRefusesRequestsNamingAHiddenBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	el := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		for {
			_, msg, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var req struct {
				ID json.RawMessage `json:"id"`
			}
			_ = json.Unmarshal(msg, &req)
			resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "from the EL"})
			if conn.Write(r.Context(), websocket.MessageText, resp) != nil {
				return
			}
		}
	}))
	defer el.Close()

	hidden := common.HexToHash("0x01")
	other := common.HexToHash("0x02")
	blocks := NewHiddenBlocks()
	blocks.Hide(hidden)
	proxy := blocks.startProxy(t, "test", "ws"+strings.TrimPrefix(el.URL, "http"))
	conn, resp, err := websocket.Dial(t.Ctx(), proxy, nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	require.NoError(gt, err)
	defer func() { _ = conn.CloseNow() }()

	refusal := `"error":{"code":-32000,"message":"block ` + hidden.Hex() + ` is hidden by the test proxy"}`
	for _, tc := range []struct {
		name    string
		request string
		want    string
	}{
		{
			name:    "forwards a request for another block",
			request: `{"jsonrpc":"2.0","id":1,"method":"debug_getRawHeader","params":["` + other.Hex() + `"]}`,
			want:    `{"id":1,"jsonrpc":"2.0","result":"from the EL"}`,
		},
		{
			name:    "forwards a request without a hash",
			request: `{"jsonrpc":"2.0","id":2,"method":"eth_getBlockByNumber","params":["0xa",false]}`,
			want:    `{"id":2,"jsonrpc":"2.0","result":"from the EL"}`,
		},
		{
			name:    "refuses a positional hash",
			request: `{"jsonrpc":"2.0","id":3,"method":"debug_getRawHeader","params":["` + hidden.Hex() + `"]}`,
			want:    `{"jsonrpc":"2.0","id":3,` + refusal + `}`,
		},
		{
			name:    "refuses an EIP-1898 block object",
			request: `{"jsonrpc":"2.0","id":4,"method":"eth_getProof","params":["0x4200000000000000000000000000000000000016",[],{"blockHash":"` + hidden.Hex() + `","requireCanonical":true}]}`,
			want:    `{"jsonrpc":"2.0","id":4,` + refusal + `}`,
		},
		{
			name:    "refuses a whole batch with named params naming it",
			request: `[{"jsonrpc":"2.0","id":5,"method":"eth_chainId","params":[]},{"jsonrpc":"2.0","id":6,"method":"eth_getLogs","params":{"filter":{"blockHash":"` + hidden.Hex() + `"}}}]`,
			want:    `[{"jsonrpc":"2.0","id":5,` + refusal + `},{"jsonrpc":"2.0","id":6,` + refusal + `}]`,
		},
	} {
		ctx, cancel := context.WithTimeout(t.Ctx(), 10*time.Second)
		require.NoError(gt, conn.Write(ctx, websocket.MessageText, []byte(tc.request)), tc.name)
		_, resp, err := conn.Read(ctx)
		cancel()
		require.NoError(gt, err, tc.name)
		require.JSONEq(gt, tc.want, string(resp), tc.name)
	}
	require.Equal(gt, int64(2), blocks.ForwardedRequests())
	require.Equal(gt, int64(3), blocks.RefusedRequests())
}
