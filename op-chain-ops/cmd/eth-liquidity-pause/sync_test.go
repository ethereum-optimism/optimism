package main

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var (
	// returnsTrueCode answers every call with the 32-byte word 1: paused() reads true and the
	// sync call succeeds.
	returnsTrueCode = common.FromHex("0x600160005260206000f3")
	// revertsCode reverts every call with empty data.
	revertsCode = common.FromHex("0x60006000fd")
)

func TestSyncChainsOnSimulatedBackends(t *testing.T) {
	pollInterval = 10 * time.Millisecond
	t.Cleanup(func() { pollInterval = 2 * time.Second })

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	messenger := common.HexToAddress("0x00000000000000000000000000000000000a11ce")
	reverting := common.HexToAddress("0x00000000000000000000000000000000000ba11e")

	l1 := simulated.NewBackend(types.GenesisAlloc{
		sender:    {Balance: new(big.Int).Mul(oneETH, big.NewInt(10))},
		messenger: {Code: returnsTrueCode},
		reverting: {Code: revertsCode},
	})
	t.Cleanup(func() { _ = l1.Close() })
	l2 := simulated.NewBackend(types.GenesisAlloc{predeploys.ETHLiquidityAddr: {Code: returnsTrueCode}})
	t.Cleanup(func() { _ = l2.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Millisecond):
				l1.Commit()
			}
		}
	}()

	l1ChainID, err := l1.Client().ChainID(ctx)
	require.NoError(t, err)
	conn := &connection{
		L1:        l1.Client(),
		L1ChainID: bigs.Uint64Strict(l1ChainID),
		Chains: []chain{{
			L2: &l2Client{URL: "simulated", ChainID: 901, Eth: l2.Client()},
			L1: &l1Contracts{Messenger: messenger},
		}},
	}
	logger := testlog.Logger(t, log.LevelInfo)

	t.Run("in sync is skipped", func(t *testing.T) {
		var out bytes.Buffer
		require.NoError(t, syncChains(ctx, logger, &out, conn, key, 10*time.Second, false))
		require.Contains(t, out.String(), "chain 901: L1 paused=true, L2 paused=true: already in sync")
		nonce, err := l1.Client().NonceAt(ctx, sender, nil)
		require.NoError(t, err)
		require.Zero(t, nonce)
	})

	t.Run("force sends the sync and waits for L2", func(t *testing.T) {
		var out bytes.Buffer
		require.NoError(t, syncChains(ctx, logger, &out, conn, key, 10*time.Second, true))
		require.Contains(t, out.String(), "sync included in L1 block")
		require.Contains(t, out.String(), "chain 901: L2 paused=true")
		nonce, err := l1.Client().NonceAt(ctx, sender, nil)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce)
	})

	t.Run("a reverting call is not sent", func(t *testing.T) {
		data, err := syncFn.EncodeArgs()
		require.NoError(t, err)
		_, err = sendTx(ctx, logger, l1.Client(), conn.L1ChainID, key, reverting, data, 10*time.Second)
		require.ErrorContains(t, err, "would revert")
		nonce, err := l1.Client().PendingNonceAt(ctx, sender)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce)
	})

	t.Run("L2 that never matches times out", func(t *testing.T) {
		err := waitL2Paused(ctx, conn.Chains[0].L2, false, 100*time.Millisecond)
		require.ErrorContains(t, err, "did not become false")
	})
}
