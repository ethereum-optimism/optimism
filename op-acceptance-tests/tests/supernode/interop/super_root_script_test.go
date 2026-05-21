package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSuperRootMigratorMatchesSupernodeAtSuppliedTimestamp verifies that when
// the SuperRootMigrator script (used by op-deployer during the Interop
// migration) is given a specific finalized timestamp, it computes the same
// super root as the supernode reports for that timestamp.
func TestSuperRootMigratorMatchesSupernodeAtSuppliedTimestamp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	finalizedTimestamp := sys.Supernode.AwaitFinalizationAdvanced()
	expected := sys.Supernode.SuperRootAt(finalizedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID())

	migrator, err := script.NewSuperRootMigrator(testlog.Logger(t, log.LevelInfo), sys.L2UserRPCURLs(), &finalizedTimestamp)
	require.NoError(t, err)
	actual, err := migrator.Run(t.Ctx())
	require.NoError(t, err)

	require.Equal(t, common.Hash(expected.Data.SuperRoot), actual,
		"migrator super root must match supernode super root at the supplied finalized timestamp")
}

// TestSuperRootMigratorSelectsLatestFinalizedTimestamp verifies that when no
// timestamp is supplied, the SuperRootMigrator selects a finalized timestamp
// (not behind the supernode's finalized timestamp at setup time) and computes
// the same super root as the supernode reports for that timestamp.
func TestSuperRootMigratorSelectsLatestFinalizedTimestamp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	finalizedAtSetup := sys.Supernode.AwaitFinalizationAdvanced()

	migrator, err := script.NewSuperRootMigrator(testlog.Logger(t, log.LevelInfo), sys.L2UserRPCURLs(), nil)
	require.NoError(t, err)
	actual, err := migrator.Run(t.Ctx())
	require.NoError(t, err)

	require.NotNil(t, migrator.TargetTimestamp, "migrator must select a finalized timestamp when none is supplied")
	require.GreaterOrEqual(t, *migrator.TargetTimestamp, finalizedAtSetup,
		"migrator-selected timestamp must not be behind the finalized timestamp observed before the migrator ran")

	expected := sys.Supernode.SuperRootAt(*migrator.TargetTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID())
	require.Equal(t, common.Hash(expected.Data.SuperRoot), actual,
		"migrator super root must match supernode super root at the migrator-selected timestamp")
}
