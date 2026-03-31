package integration_test

import (
	"context"
	"testing"

	"math/big"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/stretchr/testify/require"
)

func TestL1Genesis(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	customTimestamp := uint64(1700000000)
	customGasLimit := uint64(42_000_000)
	pragueOffset := uint64(0)

	opts, intent, st := setupGenesisChain(t, 900)
	intent.L1DevGenesisParams = &state.L1DevGenesisParams{
		BlockParams: state.L1DevGenesisBlockParams{
			Timestamp: customTimestamp,
			GasLimit:  customGasLimit,
		},
		PragueTimeOffset: &pragueOffset,
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	l1Genesis, err := inspect.L1Genesis(st)
	require.NoError(t, err)

	// Genesis allocs should contain deployed contracts.
	require.NotEmpty(t, l1Genesis.Alloc)

	// Chain ID should match intent.
	require.Equal(t, new(big.Int).SetUint64(intent.L1ChainID), l1Genesis.Config.ChainID)

	// Timestamp should match configured value.
	require.EqualValues(t, customTimestamp, l1Genesis.Timestamp)

	// Gas limit should match configured value.
	require.EqualValues(t, customGasLimit, l1Genesis.GasLimit)

	// Prague should be activated at genesis (offset 0).
	require.NotNil(t, l1Genesis.Config.PragueTime)
	require.EqualValues(t, customTimestamp, *l1Genesis.Config.PragueTime)

	// ToBlock should produce a valid block with a non-zero state root.
	block := l1Genesis.ToBlock()
	require.NotZero(t, block.Root())
	require.NotZero(t, block.Hash())

	// The state root should match what SealL1DevGenesis computed.
	require.NotNil(t, st.L1DevGenesis)
	require.NotNil(t, st.L1DevGenesis.StateHash)
	require.Equal(t, *st.L1DevGenesis.StateHash, block.Root(),
		"inspect L1Genesis state root should match SealL1DevGenesis state root")
}
