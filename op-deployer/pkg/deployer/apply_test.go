package deployer

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// The refusal must come before any stage runs: the L1 RPC URL is unreachable, so reaching the
// pipeline would fail with a different error.
func TestApplyPipelineRejectsMockSP1VerifierOnReleaseL1(t *testing.T) {
	for _, target := range []DeploymentTarget{DeploymentTargetLive, DeploymentTargetGenesis} {
		for _, l1ChainID := range []uint64{1, 11155111} {
			err := ApplyPipeline(context.Background(), ApplyPipelineOpts{
				DeploymentTarget:      target,
				DeployMockSP1Verifier: true,
				L1RPCUrl:              "http://127.0.0.1:1",
				Intent:                &state.Intent{L1ChainID: l1ChainID},
				State:                 &state.State{},
			})
			require.ErrorContains(t, err, fmt.Sprintf("L1 chain ID %d has a release SP1 verifier", l1ChainID), "target %s", target)
		}
	}
}

// TestNewPipelineEnvForgeCacheProfile checks that the forge clients run only a default-profile build,
// even where contracts may exceed the code size limit. Only a GenesisState run, which deploys to no
// chain, may run a build of another profile.
func TestNewPipelineEnvForgeCacheProfile(t *testing.T) {
	_, afs := testutil.LocalArtifacts(t)
	bundle := artifacts.Bundle{L1: afs, L2: afs}
	for _, target := range []DeploymentTarget{DeploymentTargetLive, DeploymentTargetGenesis} {
		t.Run(string(target), func(t *testing.T) {
			pEnv, err := newPipelineEnv(t.Context(), ApplyPipelineOpts{
				DeploymentTarget:          target,
				L1RPCUrl:                  "http://127.0.0.1:1",
				Logger:                    testlog.Logger(t, slog.LevelInfo),
				CacheDir:                  t.TempDir(),
				AllowUnoptimizedContracts: true,
			}, bundle)
			require.NoError(t, err)
			require.False(t, pEnv.L2Forge.AnyCacheProfile)
			require.False(t, pEnv.Forge.Client.AnyCacheProfile, "the L1 client runs live deployments")
			if target == DeploymentTargetLive {
				require.Nil(t, pEnv.Forge.Genesis, "a live deployment must not run offline")
			} else {
				require.True(t, pEnv.Forge.Genesis.NoCodeSizeLimit)
			}
		})
	}
}
