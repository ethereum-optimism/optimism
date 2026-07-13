package integration_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestApplyPipelineRustEngineGenesis exercises the production seam end-to-end: it runs the REAL
// genesis pipeline (deployer.ApplyPipeline) once with the Go engine and once with the Rust engine
// selected via ApplyPipelineOpts.ScriptEngine, and asserts the resulting L2 allocs are byte-identical.
//
// Unlike TestRustEngineL2GenesisParity (which drives the engine directly), this validates the whole
// wiring: the --script-engine selection, the pipeline.Env plumbing, rustbin binary provisioning, and
// the runL2GenesisRust branch of pipeline.GenerateL2Genesis, including CheckL2GenesisAllocs on the
// Rust dump. The L1 deploy stages run on the Go host in both cases (round 1 routes only the L2Genesis
// stage through Rust), so any difference is isolated to the L2Genesis engine.
func TestApplyPipelineRustEngineGenesis(t *testing.T) {
	// Ensure the engine is built and point the rustbin provisioner at it so the pipeline resolves the
	// exact binary this test compiled (no reliance on a stale target/ leftover).
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	runPipeline := func(t *testing.T, mode allocMode, engine env.ScriptEngineKind) *foundry.ForgeAllocs {
		t.Helper()
		lgr := testlog.Logger(t, slog.LevelWarn)
		_, pk, dk := shared.DefaultPrivkey(t)
		l1ChainID := big.NewInt(900)
		l2ChainID := uint256.NewInt(1)
		loc, _ := testutil.LocalArtifacts(t)
		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
		// Pin the CREATE2 salt so both engine runs get identical L1 deploy addresses (pipeline.init
		// generates a random salt otherwise). Only then is the L2Genesis output comparable across runs.
		st.Create2Salt = common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
		mode.configure(t, intent)

		require.NoError(t, deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
			DeploymentTarget:   deployer.DeploymentTargetGenesis,
			L1RPCUrl:           "",
			DeployerPrivateKey: pk,
			Intent:             intent,
			State:              st,
			Logger:             lgr,
			StateWriter:        pipeline.NoopStateWriter(),
			CacheDir:           testutils.IsolatedTestDirWithAutoCleanup(t),
			ScriptEngine:       engine,
		}))
		require.NotEmpty(t, st.Chains)
		require.NotNil(t, st.Chains[0].Allocs)
		return st.Chains[0].Allocs.Data
	}

	for _, mode := range allocModes(t) {
		mode := mode
		t.Run(mode.name, func(t *testing.T) {
			goDump := runPipeline(t, mode, env.ScriptEngineGo)
			rustDump := runPipeline(t, mode, env.ScriptEngineRust)

			require.Greater(t, len(goDump.Accounts), 2000, "genesis dump should be non-trivial")
			require.Equal(t, len(goDump.Accounts), len(rustDump.Accounts), "rust dump account count")
			t.Logf("[%s] pipeline L2Genesis parity: %d accounts, byte-identical (go engine vs rust engine)",
				mode.name, len(rustDump.Accounts))
			requireAllocsJSONEq(t, "pipeline-rust-vs-go", goDump, rustDump)
		})
	}
}
