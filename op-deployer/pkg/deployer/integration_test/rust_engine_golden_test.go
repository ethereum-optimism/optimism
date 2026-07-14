package integration_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// goldenCreate2Salt pins the CREATE2 salt so the L1 deploy addresses (which the L2 genesis embeds in
// predeploy storage) are stable across runs; pipeline.init randomizes the salt otherwise, which would
// make the genesis hash pins non-deterministic.
const goldenCreate2Salt = "0x00000000000000000000000000000000000000000000000000000000deadbeef"

// The *Golden tests here are the goldenized form of the corresponding *Parity tests: they run the
// Rust engine (the only engine) and pin its output against a fixture recorded from the now-deleted Go
// host at the base commit. See testdata/goldens/README.md.

// TestRustEngineL2GenesisGolden goldenizes TestRustEngineL2GenesisParity: it drives the L2Genesis
// script directly through the Rust engine (via the reconstructed L2GenesisInput) and pins the
// resulting ForgeAllocs dump by SHA-256, per alloc mode.
func TestRustEngineL2GenesisGolden(t *testing.T) {
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	for _, mode := range allocModes(t) {
		mode := mode
		t.Run(mode.name, func(t *testing.T) {
			lgr := testlog.Logger(t, slog.LevelWarn)
			_, pk, dk := shared.DefaultPrivkey(t)
			deployerAddr := crypto.PubkeyToAddress(pk.PublicKey)
			l1ChainID := big.NewInt(900)
			l2ChainID := uint256.NewInt(1)
			loc, artifactsFS := testutil.LocalArtifacts(t)
			intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
			mode.configure(t, intent)
			st.Create2Salt = common.HexToHash(goldenCreate2Salt)

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

			chainID := intent.Chains[0].ID
			input := buildL2GenesisInput(t, intent, st, chainID)

			art, err := (&foundry.ArtifactsFS{FS: artifactsFS}).ReadArtifact("L2Genesis.s.sol", "L2Genesis")
			require.NoError(t, err)
			packed, err := art.ABI.Pack("run", input)
			require.NoError(t, err)

			re, err := rustengine.Spawn(bin, rustengine.SpawnOpts{ArtifactsDir: forgeArtifactsDir(t), ChainID: 1337, Create2Deployer: true}, testWriter{t})
			require.NoError(t, err)
			defer re.Close()
			_, err = re.RunScript("L2Genesis.s.sol", "L2Genesis", packed, deployerAddr)
			require.NoError(t, err)
			require.NoError(t, re.Wipe(deployerAddr))
			rustDump, err := re.StateDump()
			require.NoError(t, err)

			// Non-vacuity guard before the hash compare: L2Genesis produces thousands of accounts.
			require.Greater(t, len(rustDump.Accounts), 2000, "L2 dump should be non-trivial")
			canonical, err := json.Marshal(rustDump)
			require.NoError(t, err)
			requireHashMatchesGolden(t, "l2genesis."+goldenName(mode.name)+".sha256", canonical)
			t.Logf("[%s] L2Genesis golden: %d accounts pinned", mode.name, len(rustDump.Accounts))
		})
	}
}

// TestApplyPipelineRustEngineGenesisGolden goldenizes TestApplyPipelineRustEngineGenesis: it runs the
// real genesis pipeline through the Rust engine and pins both the sealed L1 dev-genesis dump and the
// L2 genesis dump by SHA-256, per alloc mode.
func TestApplyPipelineRustEngineGenesisGolden(t *testing.T) {
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	for _, mode := range allocModes(t) {
		mode := mode
		t.Run(mode.name, func(t *testing.T) {
			dumps := runGenesisPipelineDefaultEngine(t, mode, nil)
			require.Greater(t, len(dumps.l2.Accounts), 2000, "L2 genesis dump should be non-trivial")
			require.Greater(t, len(dumps.l1.Accounts), 50, "L1 dev-genesis dump should be non-trivial")

			l1Canonical, err := json.Marshal(dumps.l1)
			require.NoError(t, err)
			l2Canonical, err := json.Marshal(dumps.l2)
			require.NoError(t, err)
			requireHashMatchesGolden(t, "pipeline.l1."+goldenName(mode.name)+".sha256", l1Canonical)
			requireHashMatchesGolden(t, "pipeline.l2."+goldenName(mode.name)+".sha256", l2Canonical)
			t.Logf("[%s] pipeline golden: L1 %d + L2 %d accounts pinned", mode.name, len(dumps.l1.Accounts), len(dumps.l2.Accounts))
		})
	}
}

// TestApplyPipelineRustEngineGenesisForksGolden goldenizes TestApplyPipelineRustEngineGenesisForks: it
// runs the genesis pipeline through the Rust engine once per op-e2e genesis fork and pins the L2 dump.
func TestApplyPipelineRustEngineGenesisForksGolden(t *testing.T) {
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	for _, mode := range e2eGenesisForks {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			dumps := runGenesisPipelineDefaultEngine(t, allocMode{}, forkOverrides(t, mode))
			require.Greater(t, len(dumps.l2.Accounts), 2000, "L2 genesis dump should be non-trivial")
			l2Canonical, err := json.Marshal(dumps.l2)
			require.NoError(t, err)
			requireHashMatchesGolden(t, "pipeline.fork-"+goldenName(string(mode))+".sha256", l2Canonical)
			t.Logf("[%s] L2Genesis fork golden: %d accounts pinned", mode, len(dumps.l2.Accounts))
		})
	}
}

// TestRustEngineL2SemversGolden goldenizes TestRustEngineL2SemversParity: it reads the L2 predeploy
// semvers through the Rust engine (the default) and pins them raw.
func TestRustEngineL2SemversGolden(t *testing.T) {
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

	ps, err := inspect.L2Semvers(inspect.L2SemversConfig{
		Lgr:        lgr,
		Artifacts:  artifactsFS,
		ChainState: st.Chains[0],
	})
	require.NoError(t, err)

	// Non-vacuity: the readings must be real, not empty strings that trivially match.
	require.NotEmpty(t, ps.L1Block)
	require.NotEmpty(t, ps.L2ToL1MessagePasser)
	require.NotEmpty(t, ps.OptimismMintableERC20)

	requireJSONMatchesGolden(t, "semvers.json", ps)
}

// runGenesisPipelineDefaultEngine runs a genesis ApplyPipeline on the default (Rust) engine with the
// CREATE2 salt pinned, returning the sealed L1 + L2 dumps. If overrides is non-nil it is used as the
// intent's GlobalDeployOverrides (the fork tests); otherwise mode.configure is applied.
func runGenesisPipelineDefaultEngine(t *testing.T, mode allocMode, overrides map[string]any) pipelineDumps {
	t.Helper()
	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, _ := testutil.LocalArtifacts(t)
	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
	if overrides != nil {
		intent.GlobalDeployOverrides = overrides
	} else {
		mode.configure(t, intent)
	}
	st.Create2Salt = common.HexToHash(goldenCreate2Salt)

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
	require.NotNil(t, st.L1StateDump, "L1 dev-genesis state dump should be sealed")
	return pipelineDumps{l1: st.L1StateDump.Data, l2: st.Chains[0].Allocs.Data}
}
