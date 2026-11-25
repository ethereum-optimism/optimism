package integration_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// settlementGroup sets up a genesis deployment with the OP_PRIVATE dev feature, where chain 1 is
// a settlement chain.
func settlementGroup(t *testing.T) (deployer.ApplyPipelineOpts, *state.Intent, *state.State, *devkeys.MnemonicDevKeys) {
	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
	}
	intent.Chains[0].DeployOverrides = map[string]any{"settlementChainId": 1}
	return opts, intent, st, dk
}

// addChain appends a chain with the given ID and deploy overrides to the intent.
func addChain(t *testing.T, intent *state.Intent, dk *devkeys.MnemonicDevKeys, id uint64, overrides map[string]any) *state.ChainIntent {
	chain := shared.NewChainIntent(t, dk, new(big.Int).SetUint64(devnet.DefaultChainID), uint256.NewInt(id), testCustomGasLimit)
	chain.DeployOverrides = overrides
	intent.Chains = append(intent.Chains, chain)
	return chain
}

// settlementSlot returns the settlement chain ID in the chain's ChainConfig predeploy.
func settlementSlot(t *testing.T, st *state.State, id common.Hash) *big.Int {
	chain, err := st.Chain(id)
	require.NoError(t, err)
	account, ok := chain.Allocs.Data.Accounts[predeploys.ChainConfigAddr]
	require.True(t, ok)
	return account.Storage[settlementChainIDSlot].Big()
}

// requireSharesSettlement checks that the private chain shares the settlement chain's L1
// contracts and has an L2 genesis of its own.
func requireSharesSettlement(t *testing.T, st *state.State, settlementID, privateID common.Hash) {
	t.Helper()
	settlement, err := st.Chain(settlementID)
	require.NoError(t, err)
	private, err := st.Chain(privateID)
	require.NoError(t, err)
	require.Equal(t, settlement.OpChainContracts, private.OpChainContracts)
	require.NotNil(t, private.Allocs)
}

// TestApplySettlementGroup applies a settlement chain and one of its private chains. The private
// chain gets no L1 contracts of its own: it shares the settlement chain's contracts, and its
// rollup config points at them.
func TestApplySettlementGroup(t *testing.T) {
	op_e2e.InitParallel(t)

	opts, intent, st, dk := settlementGroup(t)
	settlement := intent.Chains[0]
	private := addChain(t, intent, dk, 2, map[string]any{"settlementChainId": 1})
	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))

	requireSharesSettlement(t, st, settlement.ID, private.ID)
	require.Equal(t, int64(1), settlementSlot(t, st, settlement.ID).Int64())
	require.Equal(t, int64(1), settlementSlot(t, st, private.ID).Int64())

	_, settlementRollup, err := inspect.GenesisAndRollup(st, settlement.ID)
	require.NoError(t, err)
	_, privateRollup, err := inspect.GenesisAndRollup(st, private.ID)
	require.NoError(t, err)
	require.Equal(t, settlementRollup.L1SystemConfigAddress, privateRollup.L1SystemConfigAddress)
	require.Equal(t, settlementRollup.DepositContractAddress, privateRollup.DepositContractAddress, "both derive from the shared portal")
	require.Equal(t, int64(2), privateRollup.L2ChainID.Int64())
}

// TestApplySettlementGroupAddPrivateChain adds a private chain to an applied settlement chain and
// applies again. The new chain sets its own genesis time.
func TestApplySettlementGroupAddPrivateChain(t *testing.T) {
	op_e2e.InitParallel(t)

	opts, intent, st, dk := settlementGroup(t)
	settlement := intent.Chains[0]
	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))
	before, err := st.Chain(settlement.ID)
	require.NoError(t, err)
	contracts := before.OpChainContracts

	// The genesis target seals a new L1 genesis on every apply, which moves every chain's start
	// block. The live target keeps a start block once set.
	_, settlementRollup, err := inspect.GenesisAndRollup(st, settlement.ID)
	require.NoError(t, err)
	genesisTime := settlementRollup.Genesis.L2Time + 600
	private := addChain(t, intent, dk, 2, map[string]any{
		"settlementChainId":       1,
		"l2GenesisBlockTimestamp": hexutil.Uint64(genesisTime),
	})
	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))

	after, err := st.Chain(settlement.ID)
	require.NoError(t, err)
	require.Equal(t, contracts, after.OpChainContracts, "the settlement chain is not deployed again")
	requireSharesSettlement(t, st, settlement.ID, private.ID)
	require.Equal(t, int64(1), settlementSlot(t, st, private.ID).Int64())
	_, privateRollup, err := inspect.GenesisAndRollup(st, private.ID)
	require.NoError(t, err)
	require.Equal(t, genesisTime, privateRollup.Genesis.L2Time)
}

// TestApplySettlementGroupRejectsSettlementChainIDChange applies a public chain, then tries to
// make it a settlement chain. Its L2 genesis would keep settlementChainId 0, so apply refuses.
func TestApplySettlementGroupRejectsSettlementChainIDChange(t *testing.T) {
	op_e2e.InitParallel(t)

	opts, intent, st, dk := settlementGroup(t)
	intent.Chains[0].DeployOverrides = nil
	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))
	require.Zero(t, settlementSlot(t, st, intent.Chains[0].ID).Sign())

	// Edit a copy, as a new intent.toml would be: the state holds the applied intent itself.
	next, err := intent.Clone()
	require.NoError(t, err)
	next.Chains[0].DeployOverrides = map[string]any{"settlementChainId": 1}
	private := addChain(t, next, dk, 2, map[string]any{"settlementChainId": 1})
	opts.Intent = next
	require.ErrorContains(t, deployer.ApplyPipeline(context.Background(), opts),
		"settlementChainId changed from 0 to 1 after apply")
	_, err = st.Chain(private.ID)
	require.Error(t, err, "nothing is deployed for the private chain")
}

// TestApplySettlementGroupTwoPrivateChains applies a settlement chain with two private chains and
// a public chain.
func TestApplySettlementGroupTwoPrivateChains(t *testing.T) {
	op_e2e.InitParallel(t)

	opts, intent, st, dk := settlementGroup(t)
	settlement := intent.Chains[0]
	private2 := addChain(t, intent, dk, 2, map[string]any{"settlementChainId": 1})
	private3 := addChain(t, intent, dk, 3, map[string]any{"settlementChainId": 1})
	public := addChain(t, intent, dk, 4, nil)
	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))

	requireSharesSettlement(t, st, settlement.ID, private2.ID)
	requireSharesSettlement(t, st, settlement.ID, private3.ID)
	settlementState, err := st.Chain(settlement.ID)
	require.NoError(t, err)
	publicState, err := st.Chain(public.ID)
	require.NoError(t, err)
	require.NotEqual(t, settlementState.SystemConfigProxy, publicState.SystemConfigProxy, "the public chain deploys its own contracts")
	require.Equal(t, int64(1), settlementSlot(t, st, private2.ID).Int64())
	require.Equal(t, int64(1), settlementSlot(t, st, private3.ID).Int64())
	require.Zero(t, settlementSlot(t, st, public.ID).Sign())

	_, rollup2, err := inspect.GenesisAndRollup(st, private2.ID)
	require.NoError(t, err)
	_, rollup3, err := inspect.GenesisAndRollup(st, private3.ID)
	require.NoError(t, err)
	require.NotEqual(t, rollup2.Genesis.L2.Hash, rollup3.Genesis.L2.Hash)
}

// TestApplySettlementGroupRejectsInvalidIntents checks that apply refuses an invalid settlement
// group before it deploys anything.
func TestApplySettlementGroupRejectsInvalidIntents(t *testing.T) {
	op_e2e.InitParallel(t)

	tests := []struct {
		name   string
		mutate func(*state.Intent, *devkeys.MnemonicDevKeys)
		err    string
	}{
		{"without OP_PRIVATE", func(intent *state.Intent, dk *devkeys.MnemonicDevKeys) {
			intent.GlobalDeployOverrides = nil
			addChain(t, intent, dk, 2, map[string]any{"settlementChainId": 1})
		}, "needs the OP_PRIVATE dev feature"},
		{"global override", func(intent *state.Intent, dk *devkeys.MnemonicDevKeys) {
			intent.Chains[0].DeployOverrides = nil
			intent.GlobalDeployOverrides["settlementChainId"] = 1
			addChain(t, intent, dk, 2, nil)
		}, `globalDeployOverrides key "settlementChainId" is not allowed`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, intent, st, dk := settlementGroup(t)
			tt.mutate(intent, dk)
			require.ErrorContains(t, deployer.ApplyPipeline(context.Background(), opts), tt.err)
			require.Nil(t, st.SuperchainDeployment, "nothing is deployed")
		})
	}
}
