package pipeline

import (
	"log/slog"
	"math/big"

	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
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

func TestPrivateChainSettlement(t *testing.T) {
	intent := settlementGroupIntent()
	for i, want := range []bool{false, true, false} {
		id, private, err := PrivateChainSettlement(intent, intent.Chains[i])
		require.NoError(t, err)
		require.Equal(t, want, private, "chain %s", intent.Chains[i].ID.Big())
		if private {
			require.Equal(t, settlementID, id)
		}
	}

	intent.GlobalDeployOverrides["settlementChainId"] = 901
	_, private, err := PrivateChainSettlement(intent, intent.Chains[2])
	require.NoError(t, err)
	require.True(t, private, "a global override applies to every chain")
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
		{"settlement chain missing", func(in *state.Intent) {
			in.Chains = in.Chains[1:]
		}, "settlement chain 901 must be in the same intent"},
		{"settlement chain after its private chain", func(in *state.Intent) {
			in.Chains[0], in.Chains[1] = in.Chains[1], in.Chains[0]
		}, "settlement chain 901 must come before it"},
		{"settlement chain without its own override", func(in *state.Intent) {
			in.Chains[0].DeployOverrides = nil
		}, "chain 901 is not a settlement chain"},
		{"settlement chain is itself private", func(in *state.Intent) {
			in.Chains[2].DeployOverrides = map[string]any{"settlementChainId": 903}
			in.Chains[0].DeployOverrides["settlementChainId"] = 903
			in.Chains = []*state.ChainIntent{in.Chains[2], in.Chains[0], in.Chains[1]}
		}, "chain 901 is not a settlement chain"},
		{"private chain with a start block", func(in *state.Intent) {
			in.Chains[1].L1StartBlockHash = &common.Hash{1}
		}, "l1StartBlockHash must be unset"},
		{"private chain with alt-DA", func(in *state.Intent) {
			in.Chains[1].DangerousAltDAConfig.UseAltDA = true
		}, "alt-DA and additional dispute games must be unset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := settlementGroupIntent()
			tt.mutate(intent)
			require.ErrorContains(t, checkSettlementGroups(intent), tt.err)
		})
	}
}

func TestDeployOPChainReusesSettlementContracts(t *testing.T) {
	intent := settlementGroupIntent()
	env := &Env{Logger: testlog.Logger(t, slog.LevelInfo)}
	st := &state.State{}

	require.ErrorContains(t, DeployOPChain(env, intent, st, privateID), "settlement chain 901 of private chain 902 is not deployed")

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

func TestReuseSettlementStartBlock(t *testing.T) {
	intent := settlementGroupIntent()
	st := &state.State{}
	st.SetChainContracts(settlementID, addresses.OpChainContracts{}, true)
	st.SetChainContracts(privateID, addresses.OpChainContracts{}, true)
	st.SetChainContracts(publicID, addresses.OpChainContracts{}, true)

	reused, err := reuseSettlementStartBlock(intent, st, publicID)
	require.NoError(t, err)
	require.False(t, reused, "a public chain sets its own start block")

	_, err = reuseSettlementStartBlock(intent, st, privateID)
	require.ErrorContains(t, err, "settlement chain 901 of private chain 902 has no start block")

	genesisTime := hexutil.Uint64(1234)
	st.PinChainAnchor(settlementID, &state.L1BlockRefJSON{Hash: common.Hash{7}, Number: 42}, genesisTime)
	reused, err = reuseSettlementStartBlock(intent, st, privateID)
	require.NoError(t, err)
	require.True(t, reused)
	private, err := st.Chain(privateID)
	require.NoError(t, err)
	require.Equal(t, common.Hash{7}, private.StartBlock.Hash)
	require.Equal(t, hexutil.Uint64(42), private.StartBlock.Number)
	require.Equal(t, genesisTime, *private.GenesisTime)
}
