package node_utils

import (
	"encoding/json"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// --- Generic RPC request/response types -------------------------------------

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// push { "jsonrpc":"2.0", "method":"time", "params":{ "subscription":"0x…", "result":"…" } }
type push[Out any] struct {
	Method string `json:"method"`
	Params struct {
		SubID  uint64 `json:"subscription"`
		Result Out    `json:"result"`
	} `json:"params"`
}

// ---------------------------------------------------------------------------

func AsyncGetPrefixedWs[T any, Out any](t devtest.T, node *dsl.L2CLNode, prefix string, method string, runUntil <-chan T) <-chan Out {
	userRPC := node.Escape().UserRPC()
	wsRPC := strings.Replace(userRPC, "http", "ws", 1)

	output := make(chan Out, 128)

	go func() {
		conn, _, err := websocket.DefaultDialer.DialContext(t.Ctx(), wsRPC, nil)
		require.NoError(t, err, "dial: %v", err)
		defer conn.Close()
		defer close(output)

		// 1. send the *_subscribe request
		require.NoError(t, conn.WriteJSON(rpcRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  prefix + "_" + "subscribe_" + method,
			Params:  nil,
		}), "subscribe: %v", err)

		// 2. read the ack – blocking read just once
		var a rpcResponse
		require.NoError(t, conn.ReadJSON(&a), "ack: %v", err)
		t.Log("subscribed to websocket - id=", string(a.Result))

		// 3. defer the unsubscribe request
		defer func() {
			require.NoError(t, conn.WriteJSON(rpcRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  prefix + "_unsubscribe_" + method,
				Params:  []any{a.Result},
			}), "unsubscribe: %v", err)

			t.Log("gracefully closed websocket connection")
		}()

		// Function to handle JSON reading with error channel
		msgChan := make(chan json.RawMessage, 1) // Buffered channel to avoid goroutine leak

		go func() {
			var msg json.RawMessage
			defer close(msgChan)

			for {
				if err := conn.ReadJSON(&msg); err != nil {
					t.Log("readJSON channel closed")
					return
				}

				msgChan <- msg
			}
		}()

		// 4. start a goroutine that keeps reading pushes
		for {
			select {
			case _, ok := <-runUntil:
				// Clean‑up if necessary, then exit
				if ok {
					t.Log(method, "subscriber", "stopping: runUntil condition met")
				} else {
					t.Log(method, "subscriber", "stopping: runUntil channel closed")
				}
				return
			case <-t.Ctx().Done():
				// Clean‑up if necessary, then exit
				t.Log("unsafe head subscriber", "stopping: context cancelled")
				return
			case msg, ok := <-msgChan:
				if !ok {
					t.Log("readJSON channel closed")
					return
				}

				var p push[Out]
				require.NoError(t, json.Unmarshal(msg, &p), "decode: %v", err)

				t.Log(wsRPC, method, "received websocket message", p.Params.Result)
				output <- p.Params.Result
			}
		}

	}()

	return output
}

func GetPrefixedWs[T any, Out any](t devtest.T, node *dsl.L2CLNode, prefix string, method string, runUntil <-chan T) []Out {
	output := AsyncGetPrefixedWs[T, Out](t, node, prefix, method, runUntil)

	results := make([]Out, 0)
	for result := range output {
		results = append(results, result)
	}

	return results
}

func GetKonaWs[T any](t devtest.T, node *dsl.L2CLNode, method string, runUntil <-chan T) []eth.L2BlockRef {
	return GetPrefixedWs[T, eth.L2BlockRef](t, node, "ws", method, runUntil)
}

func GetKonaWsAsync[T any](t devtest.T, node *dsl.L2CLNode, method string, runUntil <-chan T) <-chan eth.L2BlockRef {
	return AsyncGetPrefixedWs[T, eth.L2BlockRef](t, node, "ws", method, runUntil)
}

func GetDevWS[T any](t devtest.T, node *dsl.L2CLNode, method string, runUntil <-chan T) []uint64 {
	return GetPrefixedWs[T, uint64](t, node, "dev", method, runUntil)
}

func GetDevWSAsync[T any](t devtest.T, node *dsl.L2CLNode, method string, runUntil <-chan T) <-chan uint64 {
	return AsyncGetPrefixedWs[T, uint64](t, node, "dev", method, runUntil)
}
