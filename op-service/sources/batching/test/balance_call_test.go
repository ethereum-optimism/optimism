package test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Note: These tests are in the test subpackage to avoid dependency cycles since they need to use the stubs

func TestGetBalance(t *testing.T) {
	addr := common.Address{0xab, 0xcd}
	expectedBalance := big.NewInt(248924)

	stub := NewRpcStub(t)
	stub.AddExpectedCall(NewGetBalanceCall(addr, batching.BlockLatest, expectedBalance))

	caller := batching.NewMultiCaller(stub, batching.DefaultBatchSize)
	result, err := caller.SingleCall(context.Background(), batching.BlockLatest, batching.NewBalanceCall(addr))
	require.NoError(t, err)
	require.Equal(t, expectedBalance, result.GetBigInt(0))
}
