package contracts

import (
	"context"
	"math/big"
	"testing"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	delayedWeth = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
)

func TestDelayedWeth_GetWithdrawals(t *testing.T) {
	stubRpc, weth := setupDelayedWethTest(t)
	block := rpcblock.ByNumber(482)

	addrs := []common.Address{{0x01}, {0x02}}
	expected := [][]*big.Int{
		{big.NewInt(123), big.NewInt(456)},
		{big.NewInt(123), big.NewInt(456)},
	}

	for i, addr := range addrs {
		stubRpc.SetResponse(delayedWeth, methodWithdrawals, block, []interface{}{fdgAddr, addr}, []interface{}{expected[i][0], expected[i][1]})
	}

	actual, err := weth.GetWithdrawals(context.Background(), block, fdgAddr, addrs...)
	require.NoError(t, err)
	require.Equal(t, len(expected), len(actual))
	for i := range expected {
		require.Zerof(t, expected[i][0].Cmp(actual[i].Amount), "expected: %v actual: %v", expected[i][1], actual[i].Amount)
		require.Zerof(t, expected[i][1].Cmp(actual[i].Timestamp), "expected: %v actual: %v", expected[i][0], actual[i].Timestamp)
	}
}

func TestDelayedWeth_GetBalanceAndDelay(t *testing.T) {
	stubRpc, weth := setupDelayedWethTest(t)
	block := rpcblock.ByNumber(482)
	balance := big.NewInt(23984)
	delaySeconds := int64(2983294824)
	delay := time.Duration(delaySeconds) * time.Second

	stubRpc.AddExpectedCall(batchingTest.NewGetBalanceCall(delayedWeth, block, balance))
	stubRpc.SetResponse(delayedWeth, methodDelay, block, nil, []interface{}{big.NewInt(delaySeconds)})

	actualBalance, actualDelay, err := weth.GetBalanceAndDelay(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, balance, actualBalance)
	require.Equal(t, delay, actualDelay)
}

func setupDelayedWethTest(t *testing.T) (*batchingTest.AbiBasedRpc, *DelayedWETHContract) {
	delayedWethAbi := snapshots.LoadDelayedWETHABI()
	stubRpc := batchingTest.NewAbiBasedRpc(t, delayedWeth, delayedWethAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	weth := NewDelayedWETHContract(contractMetrics.NoopContractMetrics, delayedWeth, caller)
	return stubRpc, weth
}
