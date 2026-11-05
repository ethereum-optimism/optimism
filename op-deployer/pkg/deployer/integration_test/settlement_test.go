package integration_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestApplySettlementGroup applies a settlement chain and one of its private chains. The private
// chain gets no L1 contracts of its own: it shares the settlement chain's contracts and start
// block.
func TestApplySettlementGroup(t *testing.T) {
	op_e2e.InitParallel(t)

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)

	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OPPrivateFlag),
	}
	settlement := intent.Chains[0]
	settlement.DeployOverrides = map[string]any{"settlementChainId": 1}
	private := shared.NewChainIntent(t, dk, l1ChainID, uint256.NewInt(2), testCustomGasLimit)
	private.DeployOverrides = map[string]any{"settlementChainId": 1}
	intent.Chains = append(intent.Chains, private)

	require.NoError(t, deployer.ApplyPipeline(context.Background(), opts))

	settlementState, err := st.Chain(settlement.ID)
	require.NoError(t, err)
	privateState, err := st.Chain(private.ID)
	require.NoError(t, err)
	require.Equal(t, settlementState.OpChainContracts, privateState.OpChainContracts)
	require.Equal(t, settlementState.StartBlock, privateState.StartBlock)
	require.NotNil(t, privateState.Allocs, "the private chain still gets an L2 genesis")

	_, settlementRollup, err := pipeline.RenderGenesisAndRollup(st, settlement.ID, intent)
	require.NoError(t, err)
	_, privateRollup, err := pipeline.RenderGenesisAndRollup(st, private.ID, intent)
	require.NoError(t, err)
	require.Equal(t, settlementRollup.L1SystemConfigAddress, privateRollup.L1SystemConfigAddress)
	require.Equal(t, settlementRollup.Genesis.L1, privateRollup.Genesis.L1)
	require.Equal(t, uint64(2), bigs.Uint64Strict(privateRollup.L2ChainID))
}
