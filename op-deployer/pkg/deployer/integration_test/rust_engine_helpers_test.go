package integration_test

import (
	"encoding/json"
	"io"
	"math/big"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_service "github.com/ethereum-optimism/optimism/op-service"
)

// This file holds the shared engine test helpers for the *Golden gates in this package. They were
// relocated here from the deleted rust_l2genesis_parity_test.go / rust_engine_pipeline_test.go (which
// also drove the now-removed Go script host); everything here is pure engine/JSON machinery with no
// dependency on the deleted host.

// pipelineDumps is the byte-comparable output of a single genesis ApplyPipeline run: the sealed L1
// dev-genesis allocs (from the L1 deploy stages) and the chain's L2 genesis allocs.
type pipelineDumps struct {
	l1 *foundry.ForgeAllocs
	l2 *foundry.ForgeAllocs
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
		t.Skip("no pre-built engine (RUST_BINARY_PATH_OP_SCRIPT_ENGINE) and cargo unavailable; skipping Rust engine golden test")
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
