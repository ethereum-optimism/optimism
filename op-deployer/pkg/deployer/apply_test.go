package deployer

import (
	"context"
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
func TestApplyPipelineRejectsLiveMockSP1VerifierOnReleaseL1(t *testing.T) {
	err := ApplyPipeline(context.Background(), ApplyPipelineOpts{
		DeploymentTarget:      DeploymentTargetLive,
		DeployMockSP1Verifier: true,
		L1RPCUrl:              "http://127.0.0.1:1",
		Intent:                &state.Intent{L1ChainID: 11155111},
		State:                 &state.State{},
	})
	require.ErrorContains(t, err, "L1 chain ID 11155111 has a release SP1 verifier")
}

// TestNewPipelineEnvForgeCacheProfile checks that only the L2 genesis, where contracts may exceed the
// code size limit, runs a contracts build of another profile than the default.
func TestNewPipelineEnvForgeCacheProfile(t *testing.T) {
	_, afs := testutil.LocalArtifacts(t)
	bundle := artifacts.Bundle{L1: afs, L2: afs}
	newEnv := func(t *testing.T, target DeploymentTarget, allowUnoptimized bool, anyCacheProfile bool) {
		pEnv, err := newPipelineEnv(t.Context(), ApplyPipelineOpts{
			DeploymentTarget:          target,
			L1RPCUrl:                  "http://127.0.0.1:1",
			Logger:                    testlog.Logger(t, slog.LevelInfo),
			CacheDir:                  t.TempDir(),
			AllowUnoptimizedContracts: allowUnoptimized,
		}, bundle)
		require.NoError(t, err)
		require.Equal(t, anyCacheProfile, pEnv.L2Forge.AnyCacheProfile)
		if pEnv.Forge != nil {
			require.False(t, pEnv.Forge.Client.AnyCacheProfile, "a live deployment must run only a default-profile build")
		}
	}
	t.Run("live", func(t *testing.T) { newEnv(t, DeploymentTargetLive, false, false) })
	t.Run("live, unoptimized contracts allowed", func(t *testing.T) { newEnv(t, DeploymentTargetLive, true, true) })
	t.Run("genesis", func(t *testing.T) { newEnv(t, DeploymentTargetGenesis, false, true) })
}
