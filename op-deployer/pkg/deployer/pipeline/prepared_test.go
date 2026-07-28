package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestValidatePreparedDeploymentDrift(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*state.Intent)
		wantErr string
	}{
		{
			name: "roles",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].Roles.SystemConfigOwner = common.Address{0xff}
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "fees",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].OperatorFeeScalar++
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "gas",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].GasLimit++
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "fund dev accounts",
			mutate: func(intent *state.Intent) {
				intent.FundDevAccounts = false
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "SuperchainConfig",
			mutate: func(intent *state.Intent) {
				changed := common.Address{0xee}
				intent.SuperchainConfigProxy = &changed
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "OPCM",
			mutate: func(intent *state.Intent) {
				changed := common.Address{0xdd}
				intent.OPCMAddress = &changed
			},
			wantErr: "intent OPCM address changed",
		},
		{
			name: "proof parameters",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].DeployOverrides["faultGameMaxDepth"] = uint64(99)
			},
			wantErr: "deployment intent changed",
		},
		{
			name: "L1 locator",
			mutate: func(intent *state.Intent) {
				intent.L1ContractsLocator = artifacts.MustNewFileLocator(t.TempDir())
			},
			wantErr: "L1 artifact locator changed",
		},
		{
			name: "L2 locator",
			mutate: func(intent *state.Intent) {
				intent.L2ContractsLocator = artifacts.MustNewFileLocator(t.TempDir())
			},
			wantErr: "L2 artifact locator changed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
			test.mutate(intent)
			require.ErrorContains(t, ValidatePreparedDeployment(intent, st), test.wantErr)
		})
	}
}

func TestValidatePreparedDeploymentChainSetDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*state.Intent)
	}{
		{
			name: "added",
			mutate: func(intent *state.Intent) {
				intent.Chains = append(intent.Chains, &state.ChainIntent{ID: common.Hash{0x02}})
			},
		},
		{
			name: "removed",
			mutate: func(intent *state.Intent) {
				intent.Chains = nil
			},
		},
		{
			name: "duplicated",
			mutate: func(intent *state.Intent) {
				intent.Chains = append(intent.Chains, intent.Chains[0])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
			test.mutate(intent)
			require.ErrorContains(t, ValidatePreparedDeployment(intent, st), "deployment intent changed")
		})
	}
}

func TestValidatePreparedDeploymentLateBoundPrestate(t *testing.T) {
	intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypeCannonKona)
	intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.Hash{0xaa}
	require.NoError(t, ValidatePreparedDeployment(intent, st))

	permissioned, permissionedState, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
	permissioned.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.Hash{0xbb}
	require.ErrorContains(t, ValidatePreparedDeployment(permissioned, permissionedState), "deployment intent changed")
}

func TestValidatePreparedDeploymentAfterCheckpoint(t *testing.T) {
	intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
	deployed := true
	st.Chains[0].Deployed = &deployed
	require.NoError(t, ValidatePreparedDeployment(intent, st))
}

func TestValidateCommittedPrestateOverrides(t *testing.T) {
	committed := common.Hash{0xaa}
	tests := []struct {
		name    string
		mutate  func(*state.Intent)
		wantErr string
	}{
		{
			name:   "no explicit override",
			mutate: func(*state.Intent) {},
		},
		{
			name: "matching chain override",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = committed
			},
		},
		{
			name: "changed chain override",
			mutate: func(intent *state.Intent) {
				intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.Hash{0xbb}
			},
			wantErr: "differs from the committed prestate",
		},
		{
			name: "changed global override",
			mutate: func(intent *state.Intent) {
				intent.GlobalDeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.Hash{0xcc}
			},
			wantErr: "differs from the committed prestate",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypeCannonKona)
			st.Chains[0].Prestate = committed
			test.mutate(intent)
			err := ValidateCommittedPrestateOverrides(intent, st)
			if test.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantErr)
			}
		})
	}
}

func TestPreparedChainProofParamsUsesFrozenIntent(t *testing.T) {
	intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
	chainID := intent.Chains[0].ID
	before, err := PreparedChainProofParams(st, chainID)
	require.NoError(t, err)

	intent.Chains[0].DeployOverrides["faultGameMaxDepth"] = uint64(99)
	after, err := PreparedChainProofParams(st, chainID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.NotEqual(t, uint64(99), after.DisputeMaxGameDepth)
}

func TestNewPreparedDeploymentDetachesAndSurvivesStateJSONRoundTrip(t *testing.T) {
	intent, st, _, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
	require.True(t, st.PreparedDeployment.Intent.FundDevAccounts)
	preparedJSON, err := json.Marshal(st.PreparedDeployment)
	require.NoError(t, err)

	intent.Chains[0].Roles.SystemConfigOwner = common.Address{0xaa}
	intent.Chains[0].DeployOverrides["faultGameMaxDepth"] = uint64(99)
	intent.L1ContractsLocator.URL.Path = "/changed"
	st.Chains[0].SystemConfigProxy = common.Address{0xbb}
	st.Chains[0].StartBlock.Hash = common.Hash{0xcc}
	*st.Chains[0].GenesisTime = hexutil.Uint64(99)

	afterMutationJSON, err := json.Marshal(st.PreparedDeployment)
	require.NoError(t, err)
	require.Equal(t, preparedJSON, afterMutationJSON)

	stateJSON, err := json.Marshal(st)
	require.NoError(t, err)
	var roundTripped state.State
	require.NoError(t, json.Unmarshal(stateJSON, &roundTripped))
	require.Equal(t, st.PreparedDeployment, roundTripped.PreparedDeployment)
}

func TestNewPreparedDeploymentRequiresPinnedChainTiming(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*state.ChainState)
	}{
		{
			name: "missing start block",
			mutate: func(chain *state.ChainState) {
				chain.StartBlock = nil
			},
		},
		{
			name: "missing genesis time",
			mutate: func(chain *state.ChainState) {
				chain.GenesisTime = nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent, st, bundle, _ := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
			test.mutate(st.Chains[0])
			_, err := NewPreparedDeployment(
				intent,
				st,
				st.PreparedDeployment.Deployer,
				st.PreparedDeployment.OPCM,
				bundle,
			)
			require.ErrorContains(t, err, "has no pinned anchor and genesis time")
		})
	}
}

func TestValidatePreparedArtifactContents(t *testing.T) {
	for _, target := range []struct {
		name      string
		mutate    func([2]string)
		wantError string
	}{
		{
			name: "L1",
			mutate: func(roots [2]string) {
				require.NoError(t, os.WriteFile(filepath.Join(roots[0], "artifact.json"), []byte("changed"), 0o600))
			},
			wantError: "L1 artifact contents changed",
		},
		{
			name: "L2",
			mutate: func(roots [2]string) {
				require.NoError(t, os.WriteFile(filepath.Join(roots[1], "artifact.json"), []byte("changed"), 0o600))
			},
			wantError: "L2 artifact contents changed",
		},
	} {
		t.Run(target.name, func(t *testing.T) {
			_, st, bundle, roots := preparedDeploymentFixture(t, embedded.GameTypePermissionedCannon)
			require.NoError(t, ValidatePreparedArtifactContents(st.PreparedDeployment, bundle))
			target.mutate(roots)
			require.ErrorContains(t, ValidatePreparedArtifactContents(st.PreparedDeployment, bundle), target.wantError)
		})
	}
}

func preparedDeploymentFixture(
	t *testing.T,
	gameType embedded.GameType,
) (*state.Intent, *state.State, artifacts.Bundle, [2]string) {
	t.Helper()
	chainID := common.Hash{0x01}
	opcm := common.Address{0x02}
	superchainConfig := common.Address{0x03}
	deployer := common.Address{0x04}
	l1Dir := writeArtifactBundle(t, "l1")
	l2Dir := writeArtifactBundle(t, "l2")
	chain := &state.ChainIntent{
		ID:                         chainID,
		GasLimit:                   30_000_000,
		OperatorFeeScalar:          7,
		OperatorFeeConstant:        11,
		BaseFeeVaultRecipient:      common.Address{0x10},
		L1FeeVaultRecipient:        common.Address{0x11},
		SequencerFeeVaultRecipient: common.Address{0x12},
		OperatorFeeVaultRecipient:  common.Address{0x13},
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: common.Address{0x20},
			L2ProxyAdminOwner: common.Address{0x21},
			SystemConfigOwner: common.Address{0x22},
			UnsafeBlockSigner: common.Address{0x23},
			Batcher:           common.Address{0x24},
			Proposer:          common.Address{0x25},
			Challenger:        common.Address{0x26},
		},
		DeployOverrides: map[string]any{"respectedGameType": gameType},
	}
	intent := &state.Intent{
		ConfigType:            state.IntentTypeCustom,
		L1ChainID:             1,
		OPCMAddress:           &opcm,
		SuperchainConfigProxy: &superchainConfig,
		FundDevAccounts:       true,
		L1ContractsLocator:    artifacts.MustNewFileLocator(l1Dir),
		L2ContractsLocator:    artifacts.MustNewFileLocator(l2Dir),
		Chains:                []*state.ChainIntent{chain},
		GlobalDeployOverrides: make(map[string]any),
	}
	genesisTime := hexutil.Uint64(2)
	contracts := addresses.OpChainContracts{}
	contracts.SystemConfigProxy = common.Address{0x30}
	st := &state.State{
		Create2Salt: common.Hash{0x40},
		Chains: []*state.ChainState{{
			ID:               chainID,
			OpChainContracts: contracts,
			Deployed:         new(bool),
			StartBlock:       &state.L1BlockRefJSON{Hash: common.Hash{0x50}, Number: 1, Time: 1},
			GenesisTime:      &genesisTime,
		}},
	}
	bundle := artifacts.Bundle{
		L1: foundry.OpenArtifactsDir(l1Dir).FS,
		L2: foundry.OpenArtifactsDir(l2Dir).FS,
	}
	var err error
	st.PreparedDeployment, err = NewPreparedDeployment(intent, st, deployer, opcm, bundle)
	require.NoError(t, err)
	return intent, st, bundle, [2]string{l1Dir, l2Dir}
}

func writeArtifactBundle(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "artifact.json"), []byte(contents), 0o600))
	return dir
}
