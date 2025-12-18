package opnode

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func syncConfigCliApp() *cli.App {
	syncConfigFlags := append([]cli.Flag{
		flags.L2UnsafeOnly,
		flags.SequencerEnabledFlag,
		flags.L2EngineSyncEnabled,
		flags.SyncModeFlag,
		flags.SyncModeReqRespFlag,
		flags.L2FollowSource,
		flags.L2EngineKind,
		flags.SkipSyncStartCheck,
	}, flags.P2PFlags("")..., // For p2p.sync.req-resp
	)
	return &cli.App{
		Flags: syncConfigFlags,
		Action: func(c *cli.Context) error {
			_, err := NewSyncConfig(c, log.New())
			return err
		},
	}
}

func run(args []string) error {
	return syncConfigCliApp().Run(append([]string{"test"}, args...))
}

func TestNewSyncConfigDefault(t *testing.T) {
	require.NoError(t, run(nil))
}

func TestNewSyncConfig_DerivationDisabled_NoRRSync(t *testing.T) {
	err := run([]string{
		fmt.Sprintf("--%s=true", flags.L2UnsafeOnly.Name),
		// No follow source with no derivation allowed
		fmt.Sprintf("--%s=false", flags.SyncModeReqRespFlag.Name),
	})
	require.NoError(t, err)
}

func TestNewSyncConfig_FollowSourceWithDerivationDisabled(t *testing.T) {
	err := run([]string{
		fmt.Sprintf("--%s=true", flags.L2UnsafeOnly.Name),
		fmt.Sprintf("--%s=http://example", flags.L2FollowSource.Name),
		fmt.Sprintf("--%s=false", flags.SyncModeReqRespFlag.Name),
	})
	require.NoError(t, err)
}

func TestNewSyncConfig_FollowSourceWithDerivationEnabled(t *testing.T) {
	err := run([]string{
		// unsafe-only defaults in false
		fmt.Sprintf("--%s=http://example", flags.L2FollowSource.Name),
	})
	require.ErrorContains(t, err, "cannot follow external safe/finalized with derivation enabled")
}

func TestNewSyncConfig_VerifierUnsafeOnlyWithRRSyncEnabled(t *testing.T) {
	err := run([]string{
		// verifier mode is default
		fmt.Sprintf("--%s=true", flags.L2UnsafeOnly.Name),
		fmt.Sprintf("--%s=true", flags.SyncModeReqRespFlag.Name),
	})
	require.ErrorContains(t, err, "reaching the unsafe tip would rely solely on RR sync")
}
