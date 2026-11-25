package pipeline

import (
	"math/big"
	"os"
	"path/filepath"
	"reflect"
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
	const (
		osakaHint    = " (Osaka has no entry of its own and uses prague)"
		fractionHint = " (the TOML key is updateFraction)"
		blobConfig   = "target = 6\nmax = 9\nupdateFraction = 5007716\n"
	)
	tests := []struct {
		name string
		toml string
		// keys is the reported key list, which must hold each unknown key once, without the keys below it.
		keys string
	}{
		{
			name: "osaka",
			toml: "[l1DevGenesisParams.blobSchedule.osaka]\n" + blobConfig,
			keys: "l1DevGenesisParams.blobSchedule.osaka" + osakaHint,
		},
		{
			name: "osaka as dotted keys",
			toml: "[l1DevGenesisParams.blobSchedule]\nosaka.target = 6\nosaka.max = 9\n",
			keys: "l1DevGenesisParams.blobSchedule.osaka" + osakaHint,
		},
		{
			name: "verkle",
			toml: "[l1DevGenesisParams.blobSchedule.verkle]\n" + blobConfig,
			keys: "l1DevGenesisParams.blobSchedule.verkle",
		},
		{
			name: "amsterdam",
			toml: "[l1DevGenesisParams.blobSchedule.amsterdam]\n" + blobConfig,
			keys: "l1DevGenesisParams.blobSchedule.amsterdam",
		},
		{
			name: "case-insensitive section names",
			toml: "[L1DevGenesisParams.BlobSchedule.Osaka]\n" + blobConfig,
			keys: "L1DevGenesisParams.BlobSchedule.Osaka" + osakaHint,
		},
		{
			// The decoder case-folds each key part, so the long s binds to BlobSchedule.
			name: "key part matched by unicode case folding",
			toml: "[l1DevGenesisParams.\"blob\u017fchedule\".osaka]\n" + blobConfig,
			keys: "l1DevGenesisParams.\"blob\u017fchedule\".osaka" + osakaHint,
		},
		{
			name: "unknown blob config field",
			toml: "[l1DevGenesisParams.blobSchedule.prague]\ntarget = 6\nmax = 9\nbaseFeeUpdateFraction = 5007716\n",
			keys: "l1DevGenesisParams.blobSchedule.prague.baseFeeUpdateFraction" + fractionHint,
		},
		{
			name: "array of tables lists the key once",
			toml: "[[l1DevGenesisParams.blobSchedule.verkle]]\ntarget = 6\n[[l1DevGenesisParams.blobSchedule.verkle]]\ntarget = 3\n",
			keys: "l1DevGenesisParams.blobSchedule.verkle",
		},
		{
			name: "no fork hint below a known entry",
			toml: "[l1DevGenesisParams.blobSchedule.prague]\nosaka = 1\n",
			keys: "l1DevGenesisParams.blobSchedule.prague.osaka",
		},
		{
			name: "several unknown keys",
			toml: "[l1DevGenesisParams.blobSchedule.prague]\nbaseFeeUpdateFraction = 5007716\n[l1DevGenesisParams.blobSchedule.osaka]\n" + blobConfig,
			keys: "l1DevGenesisParams.blobSchedule.prague.baseFeeUpdateFraction" + fractionHint +
				", l1DevGenesisParams.blobSchedule.osaka" + osakaHint,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			intentTOML := "configType = \"custom\"\nl1ChainID = 900\n" + tt.toml
			require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(intentTOML), 0o644))

			_, err := ReadIntent(dir)
			// The entry list is matched by its start, so that a fork added by go-ethereum does not break the test.
			require.ErrorContains(t, err, "failed to read intent file: unknown keys: "+tt.keys+"; valid blobSchedule entries are cancun, prague, bpo1, ")
			require.ErrorContains(t, err, ", each with target, max and updateFraction")
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

func TestReadIntent_RejectsUnknownKeys(t *testing.T) {
	const chain = "[[chains]]\nid = \"0x0000000000000000000000000000000000000000000000000000000000000001\"\n"
	tests := []struct {
		name string
		toml string
		// keys is the reported key list, which must hold each unknown key once, without the keys below it.
		keys string
	}{
		{
			name: "top-level key",
			toml: "useInterops = true\n",
			keys: "useInterops",
		},
		{
			name: "keys of a known table",
			toml: "[superchainRoles]\nguardian = \"0x0000000000000000000000000000000000000001\"\nproxyAdminOwner = \"0x0000000000000000000000000000000000000002\"\n",
			keys: "superchainRoles.guardian (the key is SuperchainGuardian), superchainRoles.proxyAdminOwner (the key is SuperchainProxyAdminOwner)",
		},
		{
			name: "chain key lists once for several chains",
			toml: chain + "chainFeesRecipient = \"0x0000000000000000000000000000000000000001\"\n" +
				chain + "chainFeesRecipient = \"0x0000000000000000000000000000000000000002\"\n",
			keys: "chains.chainFeesRecipient (removed, delete it)",
		},
		{
			name: "removed key in another case",
			toml: "[SuperchainRoles]\nprotocolVersionsOwner = \"0x0000000000000000000000000000000000000001\"\n",
			keys: "SuperchainRoles.protocolVersionsOwner (removed, delete it)",
		},
		{
			name: "unknown table",
			toml: "[superchain]\nguardian = 1\n[superchain.roles]\nchallenger = 2\n",
			keys: "superchain",
		},
		{
			name: "unknown inline table",
			toml: "superchain = { guardian = 1, roles = { challenger = 2 } }\n",
			keys: "superchain",
		},
		{
			name: "unknown table implied by dotted keys",
			toml: "superchain.guardian = 1\nsuperchain.roles.challenger = 2\n",
			keys: "superchain.guardian, superchain.roles.challenger",
		},
		{
			name: "key below a table that differs in case from an unknown key",
			toml: "Superchain = 1\nsuperchain.guardian = 2\n",
			keys: "Superchain, superchain.guardian",
		},
		{
			name: "key next to an override map",
			toml: chain + "deployOverride = 1\n[globalDeployOverrides]\nl2BlockTime = 1\n",
			keys: "chains.deployOverride",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			intentTOML := "configType = \"custom\"\nl1ChainID = 900\n" + tt.toml
			require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(intentTOML), 0o644))

			_, err := ReadIntent(dir)
			require.EqualError(t, err, "failed to read intent file: unknown keys: "+tt.keys)
		})
	}
}

// The override maps are free-form, and the decoder lists the keys of their nested tables as undecoded.
func TestReadIntent_AcceptsNestedOverrides(t *testing.T) {
	dir := t.TempDir()
	intentTOML := `configType = "custom"
l1ChainID = 900

[GlobalDeployOverrides]
l2BlockTime = 1
inline = { a = { b = 2 } }
[globalDeployOverrides.nested]
c = 3
[[globalDeployOverrides.list]]
d = 4

[[chains]]
id = "0x0000000000000000000000000000000000000000000000000000000000000001"
[chains.deployOverrides.customGasToken]
name = "Token"
[chains.deployOverrides.customGasToken.nested]
e = 5
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "intent.toml"), []byte(intentTOML), 0o644))

	intent, err := ReadIntent(dir)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"l2BlockTime": int64(1),
		"inline":      map[string]any{"a": map[string]any{"b": int64(2)}},
		"nested":      map[string]any{"c": int64(3)},
		"list":        []map[string]any{{"d": int64(4)}},
	}, intent.GlobalDeployOverrides)
	require.Equal(t, map[string]any{
		"customGasToken": map[string]any{"name": "Token", "nested": map[string]any{"e": int64(5)}},
	}, intent.Chains[0].DeployOverrides)
}

// ReadIntent must accept every key that op-deployer itself writes.
func TestReadIntent_AcceptsWrittenIntent(t *testing.T) {
	var intent state.Intent
	fillIntent(t, reflect.ValueOf(&intent).Elem())

	dir := t.TempDir()
	intentPath := filepath.Join(dir, "intent.toml")
	require.NoError(t, intent.WriteToFile(intentPath))
	written, err := os.ReadFile(intentPath)
	require.NoError(t, err)

	got, err := ReadIntent(dir)
	require.NoError(t, err)
	require.NoError(t, got.WriteToFile(intentPath))
	rewritten, err := os.ReadFile(intentPath)
	require.NoError(t, err)
	require.Equal(t, string(written), string(rewritten))
}

// fillIntent sets every field below v to a non-zero value, so that a field added to the intent
// type is written and read back without a change to the test.
func fillIntent(t *testing.T, v reflect.Value) {
	switch x := v.Addr().Interface().(type) {
	case *hexutil.Big:
		*x = hexutil.Big(*big.NewInt(1))
		return
	case *artifacts.Locator:
		*x = *artifacts.EmbeddedLocator
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillIntent(t, v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			require.True(t, v.Field(i).CanSet(), "unexported field %s in %s", v.Type().Field(i).Name, v.Type())
			fillIntent(t, v.Field(i))
		}
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillIntent(t, v.Index(0))
	case reflect.Array:
		fillIntent(t, v.Index(0))
	case reflect.Map:
		// The TOML library cannot write a map with a non-string key, which the prefund maps have.
		if v.Type().Key().Kind() != reflect.String {
			return
		}
		v.Set(reflect.ValueOf(map[string]any{"key": int64(1), "table": map[string]any{"key": int64(1)}}))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString("value")
	default:
		switch {
		case v.CanUint():
			v.SetUint(1)
		case v.CanInt():
			v.SetInt(1)
		default:
			require.Failf(t, "unhandled field type", "%s", v.Type())
		}
	}
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
