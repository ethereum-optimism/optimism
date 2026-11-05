package pipeline

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

var (
	settlementID = common.BigToHash(big.NewInt(901))
	privateID    = common.BigToHash(big.NewInt(902))
	publicID     = common.BigToHash(big.NewInt(903))
)

// settlementGroupIntent returns an intent with a settlement chain, one of its private chains and
// a public chain, with the OP_PRIVATE dev feature.
func settlementGroupIntent() *state.Intent {
	return &state.Intent{
		L1ContractsLocator: &artifacts.Locator{},
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
		},
		Chains: []*state.ChainIntent{
			{ID: settlementID, DeployOverrides: map[string]any{"settlementChainId": 901}},
			{ID: privateID, DeployOverrides: map[string]any{"settlementChainId": 901}},
			{ID: publicID},
		},
	}
}

func TestSettlementChainIDOverride_Accessor(t *testing.T) {
	intent := settlementGroupIntent()
	for i, want := range []uint64{901, 901, 0} {
		id, err := SettlementChainIDOverride(intent, intent.Chains[i])
		require.NoError(t, err)
		require.Equal(t, want, id)
	}

	intent.Chains[2].DeployOverrides = map[string]any{"settlementChainId": 0}
	_, private, err := PrivateChainSettlement(intent, intent.Chains[2])
	require.NoError(t, err)
	require.False(t, private, "a per-chain 0 is a public chain")
	require.NoError(t, checkSettlementGroups(intent))

	intent.GlobalDeployOverrides["SettlementChainId"] = 901
	_, err = SettlementChainIDOverride(intent, intent.Chains[2])
	require.ErrorContains(t, err, `globalDeployOverrides key "SettlementChainId" is not allowed`)
	_, _, err = calculateL2GenesisOverrides(intent, intent.Chains[2])
	require.ErrorContains(t, err, "is not allowed", "L2 genesis reads the override through the same accessor")
}

func TestPrivateChainSettlement(t *testing.T) {
	intent := settlementGroupIntent()
	for i, want := range []bool{false, true, false} {
		id, private, err := PrivateChainSettlement(intent, intent.Chains[i])
		require.NoError(t, err)
		require.Equal(t, want, private, "chain %s", intent.Chains[i].ID.Hex())
		if private {
			require.Equal(t, settlementID, id)
		}
	}
}

func TestCheckSettlementGroups(t *testing.T) {
	require.NoError(t, checkSettlementGroups(settlementGroupIntent()))

	tests := []struct {
		name   string
		mutate func(*state.Intent)
		err    string
	}{
		{"without OP_PRIVATE", func(in *state.Intent) {
			delete(in.GlobalDeployOverrides, "devFeatureBitmap")
		}, "needs the OP_PRIVATE dev feature"},
		{"global override", func(in *state.Intent) {
			in.GlobalDeployOverrides["settlementChainId"] = 901
		}, `globalDeployOverrides key "settlementChainId" is not allowed`},
		{"settlement chain missing", func(in *state.Intent) {
			in.Chains = in.Chains[1:]
		}, "settlement chain " + settlementID.Hex() + " must be in the same intent"},
		{"settlement chain after its private chain", func(in *state.Intent) {
			in.Chains[0], in.Chains[1] = in.Chains[1], in.Chains[0]
		}, "settlement chain " + settlementID.Hex() + " must come before it"},
		{"settlement chain without its own override", func(in *state.Intent) {
			in.Chains[0].DeployOverrides = nil
		}, "chain " + settlementID.Hex() + " is not a settlement chain"},
		{"settlement chain is itself private", func(in *state.Intent) {
			in.Chains[2].DeployOverrides = map[string]any{"settlementChainId": 903}
			in.Chains[0].DeployOverrides["settlementChainId"] = 903
			in.Chains = []*state.ChainIntent{in.Chains[2], in.Chains[0], in.Chains[1]}
		}, "chain " + settlementID.Hex() + " is not a settlement chain"},
		{"private chain with a start block", func(in *state.Intent) {
			in.Chains[1].L1StartBlockHash = &common.Hash{1}
		}, "l1StartBlockHash must be unset"},
		{"private chain with alt-DA", func(in *state.Intent) {
			in.Chains[1].DangerousAltDAConfig.UseAltDA = true
		}, "alt-DA and additional dispute games must be unset"},
		{"private chain with another L1 ProxyAdmin owner", func(in *state.Intent) {
			in.Chains[1].Roles.L1ProxyAdminOwner = common.Address{1}
		}, "roles.l1ProxyAdminOwner must equal that of settlement chain"},
		{"private chain with another SystemConfig owner", func(in *state.Intent) {
			in.Chains[1].Roles.SystemConfigOwner = common.Address{1}
		}, "roles.systemConfigOwner must equal that of settlement chain"},
		{"private chain with another proposer", func(in *state.Intent) {
			in.Chains[1].Roles.Proposer = common.Address{1}
		}, "roles.proposer must equal that of settlement chain"},
		{"private chain with another challenger", func(in *state.Intent) {
			in.Chains[1].Roles.Challenger = common.Address{1}
		}, "roles.challenger must equal that of settlement chain"},
		{"private chain with a custom gas token", func(in *state.Intent) {
			in.Chains[1].CustomGasToken = state.CustomGasToken{Name: "Gas", Symbol: "GAS", InitialLiquidity: (*hexutil.Big)(big.NewInt(1))}
		}, "customGasToken must equal that of settlement chain"},
		{"private chain with other dispute params", func(in *state.Intent) {
			in.Chains[1].DeployOverrides["proofMaturityDelaySeconds"] = 1
		}, "the dispute game deploy overrides must equal that of settlement chain"},
		{"private chain with a genesis timestamp", func(in *state.Intent) {
			in.Chains[1].DeployOverrides["l2GenesisBlockTimestamp"] = "0x1"
		}, `chain ` + privateID.Hex() + `: deployOverrides key "l2GenesisBlockTimestamp" is not allowed in a settlement group`},
		{"settlement chain with an L1 starting block", func(in *state.Intent) {
			in.Chains[0].DeployOverrides["l1StartingBlockTag"] = "latest"
		}, `chain ` + settlementID.Hex() + `: deployOverrides key "l1StartingBlockTag" is not allowed in a settlement group`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := settlementGroupIntent()
			tt.mutate(intent)
			require.ErrorContains(t, checkSettlementGroups(intent), tt.err)
		})
	}

	t.Run("L2 values are the private chain's own", func(t *testing.T) {
		intent := settlementGroupIntent()
		intent.Chains[1].Roles.Batcher = common.Address{1}
		intent.Chains[1].Roles.UnsafeBlockSigner = common.Address{2}
		intent.Chains[1].GasLimit = 1
		intent.Chains[1].Eip1559Elasticity = 1
		intent.Chains[1].OperatorFeeScalar = 1
		require.NoError(t, checkSettlementGroups(intent))
	})

	t.Run("a settlement chain without private chains may pin its start", func(t *testing.T) {
		intent := settlementGroupIntent()
		intent.Chains = intent.Chains[:1]
		intent.Chains[0].DeployOverrides["l1StartingBlockTag"] = "latest"
		require.NoError(t, checkSettlementGroups(intent))
	})

	t.Run("a bitmap read back from intent.toml", func(t *testing.T) {
		intent := settlementGroupIntent()
		intent.GlobalDeployOverrides["devFeatureBitmap"] = devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag).Hex()
		require.NoError(t, checkSettlementGroups(intent))
	})
}

func TestCheckSettlementChainIDsUnchanged(t *testing.T) {
	applied := settlementGroupIntent()
	applied.Chains[0].DeployOverrides = nil
	applied.Chains = applied.Chains[:1]
	st := &state.State{AppliedIntent: applied}
	st.SetChainContracts(settlementID, addresses.OpChainContracts{}, true)

	unchanged := settlementGroupIntent()
	unchanged.Chains[0].DeployOverrides = nil
	require.NoError(t, checkSettlementChainIDsUnchanged(unchanged, st))

	promoted := settlementGroupIntent()
	require.ErrorContains(t, checkSettlementChainIDsUnchanged(promoted, st),
		"chain "+settlementID.Hex()+": settlementChainId changed from 0 to 901 after apply")

	st.SetChainContracts(settlementID, addresses.OpChainContracts{}, false)
	require.NoError(t, checkSettlementChainIDsUnchanged(promoted, st), "an undeployed chain may change")
}

func TestDeployOPChainReusesSettlementContracts(t *testing.T) {
	intent := settlementGroupIntent()
	env := &Env{Logger: testlog.Logger(t, slog.LevelInfo)}
	st := &state.State{}

	require.ErrorContains(t, DeployOPChain(env, intent, st, privateID), "settlement chain "+settlementID.Hex()+" of private chain "+privateID.Hex()+" is not deployed")

	contracts := addresses.OpChainContracts{}
	contracts.OptimismPortalProxy = common.HexToAddress("0xaa")
	contracts.SystemConfigProxy = common.HexToAddress("0xbb")
	st.SetChainContracts(settlementID, contracts, true)

	require.NoError(t, DeployOPChain(env, intent, st, privateID))
	require.True(t, st.IsChainDeployed(privateID))
	private, err := st.Chain(privateID)
	require.NoError(t, err)
	require.Equal(t, contracts, private.OpChainContracts)
}

// The live strategy copies the settlement chain's start block, without an L1 client.
func TestSetStartBlockLiveStrategyPrivateChain(t *testing.T) {
	intent := settlementGroupIntent()
	env := &Env{Logger: testlog.Logger(t, slog.LevelInfo)}
	st := &state.State{}
	st.SetChainContracts(settlementID, addresses.OpChainContracts{}, true)
	st.SetChainContracts(privateID, addresses.OpChainContracts{}, true)

	require.ErrorContains(t, SetStartBlockLiveStrategy(context.Background(), intent, env, st, privateID),
		"settlement chain "+settlementID.Hex()+" of private chain "+privateID.Hex()+" has no start block")

	settlement, err := st.Chain(settlementID)
	require.NoError(t, err)
	settlement.StartBlock = &state.L1BlockRefJSON{Hash: common.Hash{9}, Number: 77, Time: 1000}
	require.NoError(t, SetStartBlockLiveStrategy(context.Background(), intent, env, st, privateID))
	private, err := st.Chain(privateID)
	require.NoError(t, err)
	require.Equal(t, *settlement.StartBlock, *private.StartBlock)
	require.NotSame(t, settlement.StartBlock, private.StartBlock)
	require.Nil(t, private.GenesisTime)
}
