package pipeline

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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

func TestResolveRenderIntent_UsesPreparedDeploymentSnapshotWhenPreparedOnly(t *testing.T) {
	_, pk, dk := shared.DefaultPrivkey(t)
	deployer := crypto.PubkeyToAddress(pk.PublicKey)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, afacts := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	superchainConfig := common.HexToAddress("0x5ea4")
	intent.SuperchainConfigProxy = &superchainConfig
	chain := intent.Chains[0]
	genesisTime := hexutil.Uint64(1_700_000_000)
	st.PinChainAnchor(chain.ID, &state.L1BlockRefJSON{
		Hash:   common.HexToHash("0xaaaa"),
		Number: 100,
		Time:   hexutil.Uint64(uint64(genesisTime) - 100),
	}, genesisTime)
	bundle := artifacts.Bundle{L1: afacts, L2: afacts}
	prepared, err := NewPreparedDeployment(intent, st, deployer, common.HexToAddress("0x1234"), bundle)
	require.NoError(t, err)
	st.PreparedDeployment = prepared

	// A live intent.toml that has since diverged from what prepare actually snapshotted.
	liveIntent, _ := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	liveIntent.FundDevAccounts = true
	dir := t.TempDir()
	require.NoError(t, liveIntent.WriteToFile(filepath.Join(dir, "intent.toml")))

	got, err := ResolveRenderIntent(dir, st)
	require.NoError(t, err)
	require.Same(t, st.PreparedDeployment.Intent, got, "must render from the frozen PreparedDeployment snapshot, not the live intent.toml")
}

func TestResolveRenderIntent_ErrorsWhenNeitherPreparedNorApplied(t *testing.T) {
	st := &state.State{Version: 1}
	_, err := ResolveRenderIntent(t.TempDir(), st)
	require.ErrorContains(t, err, "neither prepared nor applied")
}

func TestRenderGenesisAndRollup_PreparedNotApplied(t *testing.T) {
	_, intent, st, chainID := setupChainWithGenesis(t)

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

func TestRenderGenesisAndRollup_AcceptsMatchingGenesisBlockHash(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	chainState.OpChainContracts.OptimismPortalProxy = common.HexToAddress("0x1111111111111111111111111111111111111111")
	chainState.OpChainContracts.SystemConfigProxy = common.HexToAddress("0x2222222222222222222222222222222222222222")
	require.NotNil(t, chainState.GenesisBlockHash, "ComputeGenesisOutputRoot must have committed a genesis block hash")

	l2Genesis, rollupConfig, err := RenderGenesisAndRollup(st, chainID, intent)
	require.NoError(t, err)
	require.NotNil(t, l2Genesis)
	require.NotNil(t, rollupConfig)
}

func TestRenderGenesisAndRollup_RejectsGenesisBlockHashMismatch(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	chainState.OpChainContracts.OptimismPortalProxy = common.HexToAddress("0x1111111111111111111111111111111111111111")
	chainState.OpChainContracts.SystemConfigProxy = common.HexToAddress("0x2222222222222222222222222222222222222222")

	// Simulate drift between prepare and rendering: some rendering input changed since
	// prepare committed the genesis block hash, e.g. an edited (but still frozen-looking)
	// snapshot, drifted artifacts, or allocs.
	wrongHash := common.HexToHash("0xbadbad")
	chainState.GenesisBlockHash = &wrongHash

	_, _, err = RenderGenesisAndRollup(st, chainID, intent)
	require.ErrorContains(t, err, "does not match the committed GenesisBlockHash")
	require.ErrorContains(t, err, "rerun op-deployer prepare")
}

func TestRenderGenesisAndRollup_NilStartBlock(t *testing.T) {
	chainID := common.HexToHash("0x01")
	intent := &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	st := &state.State{Version: 1}
	st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)

	_, _, err := RenderGenesisAndRollup(st, chainID, intent)
	require.ErrorContains(t, err, "has no pinned start block")
}

func TestReadIntent_RejectsUnknownBlobScheduleKeys(t *testing.T) {
	tests := []struct {
		name string
		toml string
		key  string
	}{
		{
			name: "osaka",
			toml: "[l1DevGenesisParams.blobSchedule.osaka]\ntarget = 6\nmax = 9\nupdateFraction = 5007716\n",
			key:  "l1DevGenesisParams.blobSchedule.osaka",
		},
		{
			name: "verkle",
			toml: "[l1DevGenesisParams.blobSchedule.verkle]\ntarget = 6\nmax = 9\nupdateFraction = 5007716\n",
			key:  "l1DevGenesisParams.blobSchedule.verkle",
		},
		{
			name: "amsterdam",
			toml: "[l1DevGenesisParams.blobSchedule.amsterdam]\ntarget = 6\nmax = 9\nupdateFraction = 5007716\n",
			key:  "l1DevGenesisParams.blobSchedule.amsterdam",
		},
		{
			name: "case-insensitive section names",
			toml: "[L1DevGenesisParams.BlobSchedule.Osaka]\ntarget = 6\nmax = 9\nupdateFraction = 5007716\n",
			key:  "L1DevGenesisParams.BlobSchedule.Osaka",
		},
		{
			// The decoder case-folds each key part, so the long s binds to BlobSchedule.
			name: "key part matched by unicode case folding",
			toml: "[l1DevGenesisParams.\"blob\u017fchedule\".osaka]\ntarget = 6\nmax = 9\nupdateFraction = 5007716\n",
			key:  "osaka",
		},
		{
			name: "unknown blob config field",
			toml: "[l1DevGenesisParams.blobSchedule.prague]\ntarget = 6\nmax = 9\nbaseFeeUpdateFraction = 5007716\n",
			key:  "l1DevGenesisParams.blobSchedule.prague.baseFeeUpdateFraction",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			intentTOML := "configType = \"custom\"\nl1ChainID = 900\n" + tt.toml
			require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(intentTOML), 0o644))

			_, err := ReadIntent(dir)
			require.ErrorContains(t, err, tt.key)
		})
	}
}

func TestReadIntent_AcceptsKnownBlobSchedule(t *testing.T) {
	schedule := &params.BlobScheduleConfig{
		Cancun: params.DefaultCancunBlobConfig,
		Prague: params.DefaultPragueBlobConfig,
		BPO1:   &params.BlobConfig{Target: 10, Max: 15, UpdateFraction: 8346193},
	}
	intent := &state.Intent{
		ConfigType:         state.IntentTypeCustom,
		L1ChainID:          900,
		L1DevGenesisParams: &state.L1DevGenesisParams{BlobSchedule: schedule},
	}
	dir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(dir, "intent.toml")))

	got, err := ReadIntent(dir)
	require.NoError(t, err)
	require.Equal(t, schedule, got.L1DevGenesisParams.BlobSchedule)
}

// Intents in use carry keys the intent type does not know (e.g. superchainRoles.guardian).
func TestReadIntent_IgnoresUnknownKeysOutsideBlobSchedule(t *testing.T) {
	dir := t.TempDir()
	intentTOML := "configType = \"custom\"\nl1ChainID = 900\n[superchainRoles]\nguardian = \"0x0000000000000000000000000000000000000001\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(intentTOML), 0o644))

	intent, err := ReadIntent(dir)
	require.NoError(t, err)
	require.Equal(t, uint64(900), intent.L1ChainID)
}

// Every state file written before DeploymentCalldata was removed has the field, as null or as
// the calldata target's dump, so ReadState must keep ignoring unknown fields.
func TestReadState_IgnoresRemovedDeploymentCalldata(t *testing.T) {
	dir := t.TempDir()
	stateJSON := `{"version":1,"DeploymentCalldata":[{"to":"0x0000000000000000000000000000000000000001","data":"0x1234","value":"0x0"}]}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(stateJSON), 0o644))

	st, err := ReadState(dir)
	require.NoError(t, err)
	require.Equal(t, 1, st.Version)
}
