package integration_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestRustEngineL2SemversParity asserts that `op-deployer inspect l2-semvers` produces the
// identical result on the Rust engine (the default) and the Go host fallback, against the same
// freshly generated L2 genesis chain state.
func TestRustEngineL2SemversParity(t *testing.T) {
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, artifactsFS := testutil.LocalArtifacts(t)
	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)

	require.NoError(t, deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetGenesis,
		L1RPCUrl:           "",
		DeployerPrivateKey: pk,
		Intent:             intent,
		State:              st,
		Logger:             lgr,
		StateWriter:        pipeline.NoopStateWriter(),
		CacheDir:           testutils.IsolatedTestDirWithAutoCleanup(t),
	}))
	require.NotEmpty(t, st.Chains)
	require.NotNil(t, st.Chains[0].Allocs)

	run := func(engine env.ScriptEngineKind) *inspect.L2PredeploySemvers {
		ps, err := inspect.L2Semvers(inspect.L2SemversConfig{
			Lgr:          lgr,
			Artifacts:    artifactsFS,
			ChainState:   st.Chains[0],
			ScriptEngine: engine,
		})
		require.NoError(t, err)
		return ps
	}

	goSemvers := run(env.ScriptEngineGo)
	rustSemvers := run(env.ScriptEngineRust)

	// Sanity: the readings are real, not empty structs that trivially match.
	require.NotEmpty(t, goSemvers.L1Block)
	require.NotEmpty(t, goSemvers.L2ToL1MessagePasser)
	require.NotEmpty(t, goSemvers.OptimismMintableERC20)

	require.Equal(t, goSemvers, rustSemvers, "semvers must be identical across engines")
	t.Logf("L2 semvers parity: identical across Go host and Rust engine (L1Block=%s)", goSemvers.L1Block)
}
