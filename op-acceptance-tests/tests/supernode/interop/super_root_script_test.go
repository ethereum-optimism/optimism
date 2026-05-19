package interop

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const latestFinalizedLookupTimestamp = ^uint64(0)

// TestSuperRootScript_SuppliedTimestamp verifies that the SuperRootMigrator
// script (used by op-deployer during the Interop migration) computes the same
// super root as a running supernode when given a specific finalized timestamp.
func TestSuperRootScript_SuppliedTimestamp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	advanceToFinalized(t, sys)

	queryAPI := sys.Supernode.QueryAPI()
	initialResp := waitForSuperRootAtTimestamp(t, queryAPI, latestFinalizedLookupTimestamp,
		func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.NotZero(collect, resp.CurrentFinalizedTimestamp)
			require.Nil(collect, resp.Data)
		})

	targetTimestamp := initialResp.CurrentFinalizedTimestamp
	expectedChainIDs := []eth.ChainID{sys.L2A.ChainID(), sys.L2B.ChainID()}
	expected := waitForSuperRootAtTimestamp(t, queryAPI, targetTimestamp,
		func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.Equal(collect, targetTimestamp, resp.CurrentFinalizedTimestamp)
			require.NotNil(collect, resp.Data)
			for _, chainID := range expectedChainIDs {
				require.Contains(collect, resp.ChainIDs, chainID)
			}
		})
	require.NotNil(t, expected.Data)

	migrator, err := script.NewSuperRootMigrator(
		testlog.Logger(t, log.LevelInfo),
		[]string{sys.L2ELA.Escape().UserRPC(), sys.L2ELB.Escape().UserRPC()},
		&targetTimestamp,
	)
	require.NoError(t, err)
	actual, err := migrator.Run(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, common.Hash(expected.Data.SuperRoot), actual)
}

// TestSuperRootScript_LatestFinalized verifies the SuperRootMigrator picks the
// latest finalized timestamp shared by all chains when no timestamp is supplied
// and that it produces the same super root as the supernode for that timestamp.
func TestSuperRootScript_LatestFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	advanceToFinalized(t, sys)

	queryAPI := sys.Supernode.QueryAPI()
	initialResp := waitForSuperRootAtTimestamp(t, queryAPI, latestFinalizedLookupTimestamp,
		func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.NotZero(collect, resp.CurrentFinalizedTimestamp)
			require.Nil(collect, resp.Data)
		})

	finalizedTimestamp := initialResp.CurrentFinalizedTimestamp
	expectedChainIDs := []eth.ChainID{sys.L2A.ChainID(), sys.L2B.ChainID()}
	expected := waitForSuperRootAtTimestamp(t, queryAPI, finalizedTimestamp,
		func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.Equal(collect, finalizedTimestamp, resp.CurrentFinalizedTimestamp)
			require.NotNil(collect, resp.Data)
			for _, chainID := range expectedChainIDs {
				require.Contains(collect, resp.ChainIDs, chainID)
			}
		})
	require.NotNil(t, expected.Data)

	migrator, err := script.NewSuperRootMigrator(
		testlog.Logger(t, log.LevelInfo),
		[]string{sys.L2ELA.Escape().UserRPC(), sys.L2ELB.Escape().UserRPC()},
		nil,
	)
	require.NoError(t, err)
	actual, err := migrator.Run(t.Ctx())
	require.NoError(t, err)
	require.NotNil(t, migrator.TargetTimestamp)
	require.Equal(t, finalizedTimestamp, *migrator.TargetTimestamp)
	require.Equal(t, common.Hash(expected.Data.SuperRoot), actual)
}

// advanceToFinalized waits for both L2 chains to make progress, then advances
// the time-travel clock so L1 finalization (and therefore the L2 finalized
// head) catches up.
func advanceToFinalized(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
	const target = uint64(5)
	const attempts = 15
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalSafe, target, attempts),
		sys.L2BCL.ReachedFn(types.LocalSafe, target, attempts),
	)
	sys.AdvanceTime(90 * time.Second)
	sys.L1Network.WaitForFinalization()
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.Finalized, target, 30),
		sys.L2BCL.ReachedFn(types.Finalized, target, 30),
	)
}

func waitForSuperRootAtTimestamp(
	t devtest.T,
	queryAPI apis.SupernodeQueryAPI,
	timestamp uint64,
	check func(*assert.CollectT, eth.SuperRootAtTimestampResponse),
) eth.SuperRootAtTimestampResponse {
	var resp eth.SuperRootAtTimestampResponse
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var err error
		resp, err = queryAPI.SuperRootAtTimestamp(t.Ctx(), timestamp)
		require.NoError(collect, err)
		check(collect, resp)
	}, 60*time.Second, 500*time.Millisecond)

	return resp
}
