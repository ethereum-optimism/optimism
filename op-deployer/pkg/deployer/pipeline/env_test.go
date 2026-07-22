package pipeline

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestResolveRenderIntent_PrefersAppliedIntent(t *testing.T) {
	_, _, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, _ := testutil.LocalArtifacts(t)

	appliedIntent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	st.AppliedIntent = appliedIntent

	// A live intent.toml that has since diverged from what was actually applied.
	liveIntent, _ := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	liveIntent.FundDevAccounts = true
	dir := t.TempDir()
	require.NoError(t, liveIntent.WriteToFile(filepath.Join(dir, "intent.toml")))

	got, err := ResolveRenderIntent(dir, st)
	require.NoError(t, err)
	require.Same(t, appliedIntent, got, "must render from the frozen AppliedIntent, not the live intent.toml")
}

func TestResolveRenderIntent_FallsBackToLiveIntentWhenPreparedOnly(t *testing.T) {
	_, _, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, _ := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	st.Prepared = true

	dir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(dir, "intent.toml")))

	got, err := ResolveRenderIntent(dir, st)
	require.NoError(t, err)
	require.Equal(t, intent, got)
}

func TestResolveRenderIntent_ErrorsWhenNeitherPreparedNorApplied(t *testing.T) {
	st := &state.State{Version: 1}
	_, err := ResolveRenderIntent(t.TempDir(), st)
	require.ErrorContains(t, err, "neither prepared nor applied")
}

func TestRenderGenesisAndRollup_PreparedNotApplied(t *testing.T) {
	_, intent, st, chainID := setupChainWithGenesis(t)
	st.Prepared = true

	// RollupConfig requires the predicted L1 addresses prepare's predictChains would have set.
	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	chainState.OpChainContracts.OptimismPortalProxy = common.HexToAddress("0x1111111111111111111111111111111111111111")
	chainState.OpChainContracts.SystemConfigProxy = common.HexToAddress("0x2222222222222222222222222222222222222222")

	l2Genesis, rollupConfig, err := RenderGenesisAndRollup(st, chainID, intent)
	require.NoError(t, err)
	require.NotNil(t, l2Genesis)
	require.NotNil(t, rollupConfig)
}

func TestRenderGenesisAndRollup_NilStartBlock(t *testing.T) {
	chainID := common.HexToHash("0x01")
	intent := &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	st := &state.State{Version: 1}
	st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)

	_, _, err := RenderGenesisAndRollup(st, chainID, intent)
	require.ErrorContains(t, err, "has no pinned start block")
}
