package fetch

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type rpcCodeError struct {
	code int
	msg  string
}

func (e *rpcCodeError) Error() string  { return e.msg }
func (e *rpcCodeError) ErrorCode() int { return e.code }

// fakeEthAPI answers eth_call with a fixed result or error.
type fakeEthAPI struct {
	ret hexutil.Bytes
	err error
}

func (api *fakeEthAPI) Call(_ context.Context, _ map[string]any, _ rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return api.ret, api.err
}

func TestRPCL1StateCall(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantRevert bool
	}{
		{name: "revert", err: &rpcCodeError{code: 3, msg: "execution reverted: nope"}, wantRevert: true},
		{name: "revert without data", err: &rpcCodeError{code: 3, msg: "execution reverted"}, wantRevert: true},
		{name: "revert as server error", err: &rpcCodeError{code: -32000, msg: "execution reverted"}, wantRevert: true},
		{name: "capitalized revert as server error", err: &rpcCodeError{code: -32000, msg: "Execution reverted"}, wantRevert: true},
		{name: "revert mentioned in another server error", err: &rpcCodeError{code: -32000, msg: "upstream failed: execution reverted"}},
		{name: "revert message with another code", err: &rpcCodeError{code: -32603, msg: "execution reverted"}},
		{name: "missing state", err: &rpcCodeError{code: -32000, msg: "header not found"}},
		{name: "out of gas", err: &rpcCodeError{code: -32000, msg: "out of gas"}},
		{name: "rate limit", err: &rpcCodeError{code: -32005, msg: "limit exceeded"}},
		{name: "internal error", err: errors.New("boom")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l1 := dialFakeL1State(t, &fakeEthAPI{err: tt.err})
			_, err := l1.Call(context.Background(), common.Address{1}, selector("owner()"))
			require.Error(t, err)
			require.Equal(t, tt.wantRevert, errors.Is(err, errExecutionReverted), "error: %v", err)
		})
	}

	t.Run("success", func(t *testing.T) {
		l1 := dialFakeL1State(t, &fakeEthAPI{ret: hexutil.Bytes{1, 2, 3}})
		ret, err := l1.Call(context.Background(), common.Address{1}, selector("owner()"))
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3}, ret)
	})
}

func dialFakeL1State(t *testing.T, api *fakeEthAPI) *rpcL1State {
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	require.NoError(t, server.RegisterName("eth", api))
	client := ethclient.NewClient(rpc.DialInProc(server))
	t.Cleanup(client.Close)
	return &rpcL1State{client: client, block: common.Hash{1}}
}
