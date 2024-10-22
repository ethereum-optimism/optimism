package verifier

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestPendingGasLimit tests the configuration of the gas limit of the pending block,
// and if it does not conflict with the regular gas limit on the verifier or sequencer.
func TestPendingGasLimit(t *testing.T) {
	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)

	// configure the L2 gas limit to be high, and the pending gas limits to be lower for resource saving.
	cfg.DeployConfig.L2GenesisBlockGasLimit = 30_000_000
	cfg.GethOptions["sequencer"] = append(cfg.GethOptions["sequencer"], []geth.GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.Miner.GasCeil = 10_000_000
			ethCfg.Miner.RollupComputePendingBlock = true
			return nil
		},
	}...)
	cfg.GethOptions["verifier"] = append(cfg.GethOptions["verifier"], []geth.GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.Miner.GasCeil = 9_000_000
			ethCfg.Miner.RollupComputePendingBlock = true
			return nil
		},
	}...)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l2Verif := sys.NodeClient("verifier")
	l2Seq := sys.NodeClient("sequencer")

	checkGasLimit := func(client *ethclient.Client, number *big.Int, expected uint64) *types.Header {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		header, err := client.HeaderByNumber(ctx, number)
		cancel()
		require.NoError(t, err)
		require.Equal(t, expected, header.GasLimit)
		return header
	}

	// check if the gaslimits are matching the expected values,
	// and that the verifier/sequencer can use their locally configured gas limit for the pending block.
	for {
		checkGasLimit(l2Seq, big.NewInt(-1), 10_000_000)
		checkGasLimit(l2Verif, big.NewInt(-1), 9_000_000)
		checkGasLimit(l2Seq, nil, 30_000_000)
		latestVerifHeader := checkGasLimit(l2Verif, nil, 30_000_000)

		// Stop once the verifier passes genesis:
		// this implies we checked a new block from the sequencer, on both sequencer and verifier nodes.
		if latestVerifHeader.Number.Uint64() > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// TestPendingBlockIsLatest tests that we serve the latest block as pending block
func TestPendingBlockIsLatest(t *testing.T) {
	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")

	t.Run("block", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			pending, err := l2Seq.BlockByNumber(context.Background(), big.NewInt(rpc.PendingBlockNumber.Int64()))
			require.NoError(t, err)
			latest, err := l2Seq.BlockByNumber(context.Background(), nil)
			require.NoError(t, err)
			if pending.NumberU64() == latest.NumberU64() {
				require.Equal(t, pending.Hash(), latest.Hash(), "pending must exactly match latest block")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending block with same number as latest block")
	})
	t.Run("header", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			pending, err := l2Seq.HeaderByNumber(context.Background(), big.NewInt(rpc.PendingBlockNumber.Int64()))
			require.NoError(t, err)
			latest, err := l2Seq.HeaderByNumber(context.Background(), nil)
			require.NoError(t, err)
			if pending.Number.Uint64() == latest.Number.Uint64() {
				require.Equal(t, pending.Hash(), latest.Hash(), "pending must exactly match latest header")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending header with same number as latest header")
	})
}
