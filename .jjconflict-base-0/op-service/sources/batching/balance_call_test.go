package batching

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetBalance(t *testing.T) {
	addr := common.Address{0xab, 0xcd}
	expectedBalance := big.NewInt(248924)

	stub := test.NewRpcStub(t)
	stub.AddExpectedCall(test.NewGetBalanceCall(addr, rpcblock.Latest, expectedBalance))

	caller := NewMultiCaller(stub, DefaultBatchSize)
	result, err := caller.SingleCall(context.Background(), rpcblock.Latest, NewBalanceCall(addr))
	require.NoError(t, err)
	require.Equal(t, expectedBalance, result.GetBigInt(0))
}
