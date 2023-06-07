package integration_tests

import (
	"net/http"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

func TestBatching(t *testing.T) {
	config := ReadConfig("batching")

	chainIDResponse1 := `{"jsonrpc": "2.0", "result": "hello1", "id": 1}`
	chainIDResponse2 := `{"jsonrpc": "2.0", "result": "hello2", "id": 2}`
	chainIDResponse3 := `{"jsonrpc": "2.0", "result": "hello3", "id": 3}`
	netVersionResponse1 := `{"jsonrpc": "2.0", "result": "1.0", "id": 1}`
	callResponse1 := `{"jsonrpc": "2.0", "result": "ekans1", "id": 1}`

	ethAccountsResponse2 := `{"jsonrpc": "2.0", "result": [], "id": 2}`

	type mockResult struct {
		method string
		id     string
		result interface{}
	}

	chainIDMock1 := mockResult{"eth_chainId", "1", "hello1"}
	chainIDMock2 := mockResult{"eth_chainId", "2", "hello2"}
	chainIDMock3 := mockResult{"eth_chainId", "3", "hello3"}
	netVersionMock1 := mockResult{"net_version", "1", "1.0"}
	callMock1 := mockResult{"eth_call", "1", "ekans1"}

	tests := []struct {
		name                 string
		handler              http.Handler
		mocks                []mockResult
		reqs                 []*proxyd.RPCReq
		expectedRes          string
		maxUpstreamBatchSize int
		numExpectedForwards  int
	}{
		{
			name:  "backend returns batches out of order",
			mocks: []mockResult{chainIDMock1, chainIDMock2, chainIDMock3},
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "eth_chainId", nil),
				NewRPCReq("2", "eth_chainId", nil),
				NewRPCReq("3", "eth_chainId", nil),
			},
			expectedRes:          asArray(chainIDResponse1, chainIDResponse2, chainIDResponse3),
			maxUpstreamBatchSize: 2,
			numExpectedForwards:  2,
		},
		{
			// infura behavior
			name:    "backend returns single RPC response object as error",
			handler: SingleResponseHandler(500, `{"jsonrpc":"2.0","error":{"code":-32001,"message":"internal server error"},"id":1}`),
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "eth_chainId", nil),
				NewRPCReq("2", "eth_chainId", nil),
			},
			expectedRes: asArray(
				`{"error":{"code":-32011,"message":"no backends available for method"},"id":1,"jsonrpc":"2.0"}`,
				`{"error":{"code":-32011,"message":"no backends available for method"},"id":2,"jsonrpc":"2.0"}`,
			),
			maxUpstreamBatchSize: 10,
			numExpectedForwards:  1,
		},
		{
			name:    "backend returns single RPC response object for minibatches",
			handler: SingleResponseHandler(500, `{"jsonrpc":"2.0","error":{"code":-32001,"message":"internal server error"},"id":1}`),
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "eth_chainId", nil),
				NewRPCReq("2", "eth_chainId", nil),
			},
			expectedRes: asArray(
				`{"error":{"code":-32011,"message":"no backends available for method"},"id":1,"jsonrpc":"2.0"}`,
				`{"error":{"code":-32011,"message":"no backends available for method"},"id":2,"jsonrpc":"2.0"}`,
			),
			maxUpstreamBatchSize: 1,
			numExpectedForwards:  2,
		},
		{
			name: "duplicate request ids are on distinct batches",
			mocks: []mockResult{
				netVersionMock1,
				chainIDMock2,
				chainIDMock1,
				callMock1,
			},
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "net_version", nil),
				NewRPCReq("2", "eth_chainId", nil),
				NewRPCReq("1", "eth_chainId", nil),
				NewRPCReq("1", "eth_call", nil),
			},
			expectedRes:          asArray(netVersionResponse1, chainIDResponse2, chainIDResponse1, callResponse1),
			maxUpstreamBatchSize: 2,
			numExpectedForwards:  3,
		},
		{
			name:  "over max size",
			mocks: []mockResult{},
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "net_version", nil),
				NewRPCReq("2", "eth_chainId", nil),
				NewRPCReq("3", "eth_chainId", nil),
				NewRPCReq("4", "eth_call", nil),
				NewRPCReq("5", "eth_call", nil),
				NewRPCReq("6", "eth_call", nil),
			},
			expectedRes:          "{\"error\":{\"code\":-32014,\"message\":\"over batch size custom message\"},\"id\":null,\"jsonrpc\":\"2.0\"}",
			maxUpstreamBatchSize: 2,
			numExpectedForwards:  0,
		},
		{
			name: "eth_accounts does not get forwarded",
			mocks: []mockResult{
				callMock1,
			},
			reqs: []*proxyd.RPCReq{
				NewRPCReq("1", "eth_call", nil),
				NewRPCReq("2", "eth_accounts", nil),
			},
			expectedRes:          asArray(callResponse1, ethAccountsResponse2),
			maxUpstreamBatchSize: 2,
			numExpectedForwards:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Server.MaxUpstreamBatchSize = tt.maxUpstreamBatchSize

			handler := tt.handler
			if handler == nil {
				router := NewBatchRPCResponseRouter()
				for _, mock := range tt.mocks {
					router.SetRoute(mock.method, mock.id, mock.result)
				}
				handler = router
			}

			goodBackend := NewMockBackend(handler)
			defer goodBackend.Close()
			require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

			client := NewProxydClient("http://127.0.0.1:8545")
			_, shutdown, err := proxyd.Start(config)
			require.NoError(t, err)
			defer shutdown()

			res, statusCode, err := client.SendBatchRPC(tt.reqs...)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, statusCode)
			RequireEqualJSON(t, []byte(tt.expectedRes), res)

			if tt.numExpectedForwards != 0 {
				require.Equal(t, tt.numExpectedForwards, len(goodBackend.Requests()))
			}

			if handler, ok := handler.(*BatchRPCResponseRouter); ok {
				for i, mock := range tt.mocks {
					require.Equal(t, 1, handler.GetNumCalls(mock.method, mock.id), i)
				}
			}
		})
	}
}
