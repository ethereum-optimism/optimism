package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSuperRootScript(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})

	system.FinalizeL1()

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})

	actors := system.Actors

	clients := map[string]*ethclient.Client{
		"chainA": actors.ChainA.SequencerEngine.EthClient(),
		"chainB": actors.ChainB.SequencerEngine.EthClient(),
	}

	gt.Run("SuppliedTimestamp", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		safeTime := actors.ChainA.Sequencer.L2Safe().Time
		expected, err := actors.Supervisor.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(safeTime))
		require.NoError(t, err)
		migrator, err := script.NewSuperRootMigratorWithClients(testlog.Logger(t, log.LevelInfo), clients, &safeTime)
		require.NoError(t, err)
		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(expected.SuperRoot), actual)
	})

	gt.Run("LatestFinalized", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)

		syncStatus, err := actors.Supervisor.SyncStatus(t.Ctx())
		require.NoError(t, err)
		finalizedTime := syncStatus.FinalizedTimestamp
		expected, err := actors.Supervisor.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(finalizedTime))
		require.NoError(t, err)

		migrator, err := script.NewSuperRootMigratorWithClients(testlog.Logger(t, log.LevelInfo), clients, &finalizedTime)
		require.NoError(t, err)
		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(expected.SuperRoot), actual)
	})
}
