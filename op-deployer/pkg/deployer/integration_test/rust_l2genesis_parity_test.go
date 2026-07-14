package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestRustEngineL2GenesisParity is the L2Genesis milestone gate for the Rust script engine
// (op-geth decoupling spike). For each L2Genesis alloc mode it:
//  1. runs the real genesis pipeline (the authoritative Go reference dump);
//  2. reconstructs the exact L2GenesisInput the pipeline fed the L2Genesis script, validated by
//     running the same input through a fresh Go host and asserting byte-parity with the reference;
//  3. replays the identical ABI-packed run() calldata through the Rust engine and asserts the Rust
//     ForgeAllocs dump is byte-identical to the reference.
//
// The "default" subtest is the milestone gate; the other modes exercise the extra cheat surface
// (custom gas token, interop, l2cm dev features) for stronger coverage.
func TestRustEngineL2GenesisParity(t *testing.T) {
	bin := buildRustScriptEngine(t)

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

			require.NoError(t, deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
				DeploymentTarget:   deployer.DeploymentTargetGenesis,
				L1RPCUrl:           "",
				DeployerPrivateKey: pk,
				Intent:             intent,
				State:              st,
				Logger:             lgr,
				StateWriter:        pipeline.NoopStateWriter(),
				CacheDir:           testutils.IsolatedTestDirWithAutoCleanup(t),
				// Pin the reference dump to the Go engine so this stays a Go-vs-Rust parity check even
				// though the default engine is now Rust.
				ScriptEngine: env.ScriptEngineGo,
			}))
			require.NotEmpty(t, st.Chains)
			require.NotNil(t, st.Chains[0].Allocs)
			referenceDump := st.Chains[0].Allocs.Data

			chainID := intent.Chains[0].ID
			input := buildL2GenesisInput(t, intent, st, chainID)

			// Leg A: a fresh Go host with the reconstructed input must reproduce the reference
			// dump. This proves the reconstruction is exact, so the Rust comparison is meaningful.
			goDump := runGoL2Genesis(t, lgr, deployerAddr, artifactsFS, input)
			requireAllocsJSONEq(t, "go-manual-vs-reference", referenceDump, goDump)

			// Leg B: the identical run() calldata through the Rust engine.
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

			// Guard against a trivial (empty) match: L2Genesis produces thousands of accounts.
			require.Greater(t, len(referenceDump.Accounts), 2000, "reference dump should be non-trivial")
			require.Equal(t, len(referenceDump.Accounts), len(rustDump.Accounts), "rust dump account count")
			t.Logf("[%s] L2Genesis parity: %d accounts, byte-identical across Go host and Rust engine",
				mode.name, len(rustDump.Accounts))

			requireAllocsJSONEq(t, "rust-vs-reference", referenceDump, rustDump)
		})
	}
}

// runGoL2Genesis mirrors pipeline.GenerateL2Genesis lines 65-133 (minus validation): build a
// non-forked DefaultScriptHost, run the L2Genesis script, wipe the deployer, and dump.
func runGoL2Genesis(t *testing.T, lgr log.Logger, deployer common.Address, artifactsFS foundry.StatDirFs, input opcm.L2GenesisInput) *foundry.ForgeAllocs {
	t.Helper()
	host, err := env.DefaultScriptHost(broadcaster.NoopBroadcaster(), lgr, deployer, artifactsFS)
	require.NoError(t, err)
	scr, err := opcm.NewL2GenesisScript(host)
	require.NoError(t, err)
	require.NoError(t, scr.Run(input))
	host.Wipe(deployer)
	dump, err := host.StateDump()
	require.NoError(t, err)
	return dump
}

// buildL2GenesisInput reconstructs the L2GenesisInput that pipeline.GenerateL2Genesis builds for a
// given mode, reading the chain intent/state that the pipeline populated. It mirrors the pipeline's
// default overrides (no global/deploy override keys other than devFeatureBitmap are used by the
// test modes), buildCGTConfig, and buildDevFeatureBitmap.
func buildL2GenesisInput(t *testing.T, intent *state.Intent, st *state.State, chainID common.Hash) opcm.L2GenesisInput {
	t.Helper()
	thisIntent, err := intent.Chain(chainID)
	require.NoError(t, err)
	thisChainState, err := st.Chain(chainID)
	require.NoError(t, err)

	schedule := standard.DefaultHardforkSchedule()
	vaultMin := func() *big.Int { return standard.VaultMinWithdrawalAmount.ToInt() }
	localNet := genesis.WithdrawalNetwork("local")
	localWd := big.NewInt(int64(localNet.ToUint8()))

	// buildCGTConfig
	cgtName, cgtSymbol := "", ""
	nativeLiquidity := big.NewInt(0)
	var liquidityOwner common.Address
	if thisIntent.IsCustomGasTokenEnabled() {
		cgtName = thisIntent.CustomGasToken.Name
		cgtSymbol = thisIntent.CustomGasToken.Symbol
		nativeLiquidity = thisIntent.GetInitialLiquidity()
		liquidityOwner = thisIntent.GetLiquidityControllerOwner()
	}

	// buildDevFeatureBitmap
	var devFeatureBitmap common.Hash
	switch v := intent.GlobalDeployOverrides["devFeatureBitmap"].(type) {
	case common.Hash:
		devFeatureBitmap = v
	case string:
		devFeatureBitmap = common.HexToHash(v)
	}

	return opcm.L2GenesisInput{
		L1ChainID:                                new(big.Int).SetUint64(intent.L1ChainID),
		L2ChainID:                                chainID.Big(),
		L1CrossDomainMessengerProxy:              thisChainState.L1CrossDomainMessengerProxy,
		L1StandardBridgeProxy:                    thisChainState.L1StandardBridgeProxy,
		L1ERC721BridgeProxy:                      thisChainState.L1Erc721BridgeProxy,
		OpChainProxyAdminOwner:                   thisIntent.Roles.L2ProxyAdminOwner,
		BaseFeeVaultWithdrawalNetwork:            localWd,
		L1FeeVaultWithdrawalNetwork:              localWd,
		SequencerFeeVaultWithdrawalNetwork:       localWd,
		OperatorFeeVaultWithdrawalNetwork:        localWd,
		SequencerFeeVaultMinimumWithdrawalAmount: vaultMin(),
		BaseFeeVaultMinimumWithdrawalAmount:      vaultMin(),
		L1FeeVaultMinimumWithdrawalAmount:        vaultMin(),
		OperatorFeeVaultMinimumWithdrawalAmount:  vaultMin(),
		BaseFeeVaultRecipient:                    thisIntent.BaseFeeVaultRecipient,
		L1FeeVaultRecipient:                      thisIntent.L1FeeVaultRecipient,
		SequencerFeeVaultRecipient:               thisIntent.SequencerFeeVaultRecipient,
		OperatorFeeVaultRecipient:                thisIntent.OperatorFeeVaultRecipient,
		GovernanceTokenOwner:                     standard.GovernanceTokenOwner,
		Fork:                                     big.NewInt(schedule.SolidityForkNumber(1)),
		EnableGovernance:                         false,
		FundDevAccounts:                          intent.FundDevAccounts,
		UseCustomGasToken:                        thisIntent.IsCustomGasTokenEnabled(),
		UseInterop:                               intent.UseInterop,
		GasPayingTokenName:                       cgtName,
		GasPayingTokenSymbol:                     cgtSymbol,
		NativeAssetLiquidityAmount:               nativeLiquidity,
		LiquidityControllerOwner:                 liquidityOwner,
		DevFeatureBitmap:                         devFeatureBitmap,
	}
}

func requireAllocsJSONEq(t *testing.T, label string, want, got *foundry.ForgeAllocs) {
	t.Helper()
	wb, err := json.Marshal(want)
	require.NoError(t, err)
	gb, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, string(wb), string(gb), "state dump mismatch: %s", label)
}

// buildRustScriptEngine returns the Rust engine binary path. It prefers a pre-built binary named by
// RUST_BINARY_PATH_OP_SCRIPT_ENGINE (how CI supplies it to the cargo-less Go executors); otherwise
// it cargo-builds locally, and skips only when neither a pre-built binary nor cargo is available.
func buildRustScriptEngine(t *testing.T) string {
	if p, ok := rustengine.PrebuiltEngineBinary(); ok {
		return p
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("no pre-built engine (RUST_BINARY_PATH_OP_SCRIPT_ENGINE) and cargo unavailable; skipping Rust engine parity test")
	}
	rustDir := filepath.Join(monorepoRoot(t), "rust")
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = rustDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cargo build op-script-engine failed: %v\n%s", err, out)
	}
	return filepath.Join(rustDir, "target", "debug", "op-script-engine")
}

func forgeArtifactsDir(t *testing.T) string {
	return filepath.Join(monorepoRoot(t), "packages", "contracts-bedrock", "forge-artifacts")
}

func monorepoRoot(t *testing.T) string {
	_, testFilename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root, err := op_service.FindMonorepoRoot(testFilename)
	require.NoError(t, err)
	return root
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("[engine] %s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

var _ io.Writer = testWriter{}
