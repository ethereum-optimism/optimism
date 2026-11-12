package forge

import (
	"context"
	"log/slog"
	"math/big"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
)

func TestStartFork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	srcRPC, src := devnet.DefaultAnvilRPC(t, testlog.Logger(t, slog.LevelInfo))
	anvilPath, err := exec.LookPath("anvil")
	require.NoError(t, err)

	fork, err := StartFork(ctx, StaticBinary(anvilPath), srcRPC, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fork.Stop()) })
	forked, err := ethclient.DialContext(ctx, fork.RPCURL())
	require.NoError(t, err)

	chainID, err := forked.ChainID(ctx)
	require.NoError(t, err)
	require.EqualValues(t, devnet.DefaultChainID, bigs.Uint64Strict(chainID))

	// A funded source account sends on the fork without its key.
	from := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	to := common.Address{0x0f}
	var txHash common.Hash
	require.NoError(t, forked.Client().CallContext(ctx, &txHash, "eth_sendTransaction", map[string]any{
		"from":  from,
		"to":    to,
		"value": (*hexutil.Big)(big.NewInt(1)),
	}))
	receipt, err := bind.WaitMined(ctx, forked, txHash)
	require.NoError(t, err)
	require.EqualValues(t, types.ReceiptStatusSuccessful, receipt.Status)
	balance, err := forked.BalanceAt(ctx, to, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, bigs.Uint64Strict(balance))

	balance, err = src.BalanceAt(ctx, to, nil)
	require.NoError(t, err)
	require.Zero(t, bigs.Uint64Strict(balance), "the fork must not change the source chain")
}

// TestStartFork_ReportsAnvilError checks that an anvil exit fails StartFork at once, with its output.
func TestStartFork_ReportsAnvilError(t *testing.T) {
	anvilPath, err := exec.LookPath("anvil")
	require.NoError(t, err)

	start := time.Now()
	_, err = StartFork(context.Background(), StaticBinary(anvilPath), "http://127.0.0.1:1", 0)
	require.Error(t, err)
	require.Less(t, time.Since(start), 30*time.Second)
	require.ErrorContains(t, err, "anvil exited")
	require.ErrorContains(t, err, "Error:", "the error includes the output of anvil")
}
