package pipeline

import (
	"encoding/json"
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
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestReadIntentRejectsLegacyAltDAConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(`
[[chains]]

[chains.dangerousAltDAConfig]
useAltDA = true
`), 0o600))

	_, err := ReadIntent(dir)
	require.ErrorIs(t, err, state.ErrAltDANoLongerSupported)
}

func TestReadIntentRejectsAltDADeployOverrides(t *testing.T) {
	tests := []struct {
		name   string
		intent string
	}{
		{
			name: "global override",
			intent: `
[globalDeployOverrides]
USEALTDA = false
`,
		},
		{
			name: "chain override",
			intent: `
[[chains]]

[chains.deployOverrides]
DaChAlLeNgEpRoXy = "0x0000000000000000000000000000000000000000"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(tt.intent), 0o600))

			_, err := ReadIntent(dir)
			require.ErrorIs(t, err, state.ErrAltDANoLongerSupported)
		})
	}
}

func TestReadStateRejectsLegacyAltDAChallengeAddress(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{
  "opChainDeployments": [{
    "id": "0x0000000000000000000000000000000000000000000000000000000000000001",
    "AltDAChallengeProxy": "0x1111111111111111111111111111111111111111"
  }]
}`), 0o600))

	_, err := ReadState(dir)
	require.ErrorIs(t, err, state.ErrAltDANoLongerSupported)
}

func TestReadStateRejectsLegacyAltDAConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{
  "appliedIntent": {
    "chains": [{
      "dangerousAltDAConfig": {
        "useAltDA": true
      }
    }]
  }
}`), 0o600))

	_, err := ReadState(dir)
	require.ErrorIs(t, err, state.ErrAltDANoLongerSupported)
}

func TestReadStateRejectsAltDADeployOverrides(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{
			name: "applied global override",
			state: `{
  "appliedIntent": {
    "globalDeployOverrides": {
      "DaBoNdSiZe": 0
    }
  }
}`,
		},
		{
			name: "prepared chain override",
			state: `{
  "preparedDeployment": {
    "intent": {
      "chains": [{
        "deployOverrides": {
          "DARESOLVEWINDOW": 0
        }
      }]
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(tt.state), 0o600))

			_, err := ReadState(dir)
			require.ErrorIs(t, err, state.ErrAltDANoLongerSupported)
		})
	}
}

func TestReadStateDropsEmptyLegacyAltDAFields(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{
  "appliedIntent": {
    "chains": [{
      "dangerousAltDAConfig": {
        "useAltDA": false
      }
    }]
  },
  "preparedDeployment": {
    "intent": {
      "chains": [{
        "dangerousAltDAConfig": {}
      }]
    },
    "chains": [{
      "AltDAChallengeProxy": "0x0000000000000000000000000000000000000000",
      "AltDAChallengeImpl": "0x0000000000000000000000000000000000000000"
    }]
  },
  "opChainDeployments": [{
    "AltDAChallengeProxy": "0x0000000000000000000000000000000000000000",
    "AltDAChallengeImpl": "0x0000000000000000000000000000000000000000"
  }]
}`), 0o600))

	st, err := ReadState(dir)
	require.NoError(t, err)
	encoded, err := json.Marshal(st)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "dangerousAltDAConfig")
	require.NotContains(t, string(encoded), "AltDAChallenge")
}

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
