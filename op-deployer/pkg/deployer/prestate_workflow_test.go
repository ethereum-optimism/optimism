package deployer

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPrestateWorkflowFromPrepareChains(t *testing.T) {
	tests := []struct {
		name               string
		permissionlessType embedded.GameType
		selected           common.Hash
		fallback           common.Hash
		wantSelected       common.Hash
		wantCannonFallback common.Hash
	}{
		{
			name:               "permissioned and CANNON_KONA",
			permissionlessType: embedded.GameTypeCannonKona,
			selected:           common.HexToHash("0x11"),
			fallback:           common.HexToHash("0x22"),
			wantSelected:       common.HexToHash("0x11"),
			wantCannonFallback: common.HexToHash("0x22"),
		},
		{
			name:               "permissioned and SUPER_CANNON_KONA",
			permissionlessType: embedded.GameTypeSuperCannonKona,
			selected:           common.HexToHash("0x33"),
			wantSelected:       common.HexToHash("0x33"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployedID := common.HexToHash("0x01")
			permissionedID := common.HexToHash("0x02")
			permissionlessID := common.HexToHash("0x03")
			chainIDs := []common.Hash{deployedID, permissionedID, permissionlessID}
			intent := newPrestateWorkflowIntent(t, chainIDs)
			intent.Chains[2].DeployOverrides = map[string]any{"respectedGameType": tt.permissionlessType}

			historicalSelected := common.HexToHash("0xaa")
			historicalFallback := common.HexToHash("0xbb")
			st := &state.State{
				Version:     1,
				Prepared:    true,
				Create2Salt: common.HexToHash("0xcc"),
			}
			var deployedContracts addresses.OpChainContracts
			deployedContracts.SystemConfigProxy = common.HexToAddress("0x00000000000000000000000000000000000000dd")
			st.SetChainContracts(deployedID, deployedContracts, true)
			deployed, err := st.Chain(deployedID)
			require.NoError(t, err)
			deployed.Prestate = historicalSelected
			deployed.CannonFallbackPrestate = historicalFallback

			predicted := make(map[common.Hash]common.Address)
			run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
				require.NotNil(t, st.InteropDepSet)
				require.Len(t, st.InteropDepSet.Chains(), len(chainIDs))
				for _, chainID := range chainIDs {
					require.True(t, st.InteropDepSet.HasChain(eth.ChainIDFromBytes32(chainID)))
				}

				chainID := common.BigToHash(in.L2ChainId)
				systemConfigProxy := common.BytesToAddress(chainID.Bytes())
				predicted[chainID] = systemConfigProxy
				out := emptyDeployOPChainOutput()
				out.SystemConfigProxy = systemConfigProxy
				return out, nil
			}

			require.NoError(t, prepareChains(testlog.Logger(t, slog.LevelInfo), intent, st, run))
			require.Len(t, predicted, 2)
			require.NotContains(t, predicted, deployedID)
			require.Len(t, st.Chains, len(chainIDs))
			for _, chainID := range chainIDs {
				_, err := st.Chain(chainID)
				require.NoError(t, err)
			}

			workdir := t.TempDir()
			require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
			require.NoError(t, pipeline.WriteState(workdir, st))

			cfg := PrestateConfig{
				Workdir:     workdir,
				Logger:      testlog.Logger(t, slog.LevelInfo),
				Prestate:    tt.selected.Hex(),
				PrestateSet: true,
			}
			if tt.fallback != (common.Hash{}) {
				cfg.CannonFallbackPrestate = tt.fallback.Hex()
				cfg.CannonFallbackPrestateSet = true
			}
			require.NoError(t, Prestate(context.Background(), cfg))

			persisted, err := pipeline.ReadState(workdir)
			require.NoError(t, err)
			require.True(t, persisted.Prepared)
			require.NotNil(t, persisted.InteropDepSet)
			for _, chainID := range chainIDs {
				require.True(t, persisted.InteropDepSet.HasChain(eth.ChainIDFromBytes32(chainID)))
			}

			deployed, err = persisted.Chain(deployedID)
			require.NoError(t, err)
			require.Equal(t, historicalSelected, deployed.Prestate)
			require.Equal(t, historicalFallback, deployed.CannonFallbackPrestate)

			permissioned, err := persisted.Chain(permissionedID)
			require.NoError(t, err)
			require.Equal(t, predicted[permissionedID], permissioned.SystemConfigProxy)
			require.Zero(t, permissioned.Prestate)
			require.Zero(t, permissioned.CannonFallbackPrestate)

			permissionless, err := persisted.Chain(permissionlessID)
			require.NoError(t, err)
			require.Equal(t, predicted[permissionlessID], permissionless.SystemConfigProxy)
			require.Equal(t, tt.wantSelected, permissionless.Prestate)
			require.Equal(t, tt.wantCannonFallback, permissionless.CannonFallbackPrestate)
		})
	}
}

func newPrestateWorkflowIntent(t *testing.T, chainIDs []common.Hash) *state.Intent {
	t.Helper()

	intent, err := state.NewIntentCustom(1, chainIDs)
	require.NoError(t, err)
	opcmAddr := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	superchainConfigProxy := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	intent.OPCMAddress = &opcmAddr
	intent.SuperchainConfigProxy = &superchainConfigProxy
	intent.SuperchainRoles = nil
	intent.GlobalDeployOverrides = make(map[string]any)

	role := common.HexToAddress("0x0000000000000000000000000000000000000001")
	for _, chain := range intent.Chains {
		chain.BaseFeeVaultRecipient = role
		chain.L1FeeVaultRecipient = role
		chain.SequencerFeeVaultRecipient = role
		chain.OperatorFeeVaultRecipient = role
		chain.Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
		chain.Eip1559Denominator = standard.Eip1559Denominator
		chain.Eip1559Elasticity = standard.Eip1559Elasticity
		chain.Roles = state.ChainRoles{
			L1ProxyAdminOwner: role,
			L2ProxyAdminOwner: role,
			SystemConfigOwner: role,
			UnsafeBlockSigner: role,
			Batcher:           role,
			Proposer:          role,
			Challenger:        role,
		}
		chain.DeployOverrides = make(map[string]any)
	}
	require.NoError(t, intent.Check())
	return &intent
}
