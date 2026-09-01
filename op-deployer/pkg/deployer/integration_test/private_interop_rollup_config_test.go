package integration_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	projectiongenesis "github.com/ethereum-optimism/optimism/op-private-interop/genesis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Op-deployer emits one private-chain genesis and rollup config. The public projection is a pure
// local derivation from those artifacts, not a second deployer role or apply run.
func TestPrivateInteropRollupConfigAndProjection(t *testing.T) {
	op_e2e.InitParallel(t)

	_, applied := generatePrivateInteropPair(t)
	privateChainGenesis, privateChainRollup, err := pipeline.RenderGenesisAndRollup(
		applied.st, applied.intent.Chains[0].ID, applied.intent,
	)
	require.NoError(t, err)
	require.NoError(t, privateChainRollup.Check())
	require.NotNil(t, privateChainRollup.PrivateInterop, "rollup marker activates projection consumers")

	publicProjectionGenesis, err := projectiongenesis.ProjectGenesisFrom(privateChainGenesis)
	require.NoError(t, err)
	publicProjectionRollup, err := projectiongenesis.ProjectRollupConfigFrom(privateChainRollup, publicProjectionGenesis)
	require.NoError(t, err)
	require.NoError(t, publicProjectionRollup.Check())

	require.Zero(t, publicProjectionRollup.L2ChainID.Cmp(privateChainRollup.L2ChainID))
	require.NotEqual(t, privateChainRollup.Genesis.L2.Hash, publicProjectionRollup.Genesis.L2.Hash)
	require.Equal(t, uint64(gethparams.MaxGasLimit), publicProjectionRollup.Genesis.SystemConfig.GasLimit)
	scalars, err := eth.DecodeScalar(publicProjectionRollup.Genesis.SystemConfig.Scalar)
	require.NoError(t, err)
	require.Zero(t, scalars.BaseFeeScalar)
	require.Zero(t, scalars.BlobBaseFeeScalar)
	require.Equal(t, eth.OperatorFeeParams{}, publicProjectionRollup.Genesis.SystemConfig.OperatorFee())
	require.NotNil(t, publicProjectionRollup.PrivateInterop, "the marker remains available after projection")
	require.False(t, devfeatures.IsDevFeatureEnabled(
		publicProjectionGenesis.Alloc[predeploys.L2DevFeatureFlagsAddr].Storage[l2DevFeatureBitmapSlot],
		devfeatures.PrivateInteropFlag,
	), "the projected chain must not expose the private-chain feature on L2")

	_, artifactsFS := testutil.LocalArtifacts(t)
	for _, expected := range []struct {
		proxy    common.Address
		file     string
		contract string
	}{
		{predeploys.L1BlockAddr, "L1Block.sol", "L1Block"},
		{predeploys.L2ToL1MessagePasserAddr, "L2ToL1MessagePasser.sol", "L2ToL1MessagePasser"},
		{predeploys.L2toL2CrossDomainMessengerAddr, "L2ToL2CrossDomainMessengerReplay.sol", "L2ToL2CrossDomainMessengerReplay"},
		{predeploys.SuperchainETHBridgeAddr, "SuperchainETHBridge.sol", "SuperchainETHBridge"},
		{predeploys.ETHLiquidityAddr, "ETHLiquidity.sol", "ETHLiquidity"},
		{predeploys.ClaimRegistryAddr, "ClaimRegistry.sol", "ClaimRegistry"},
		{predeploys.EventReplayerAddr, "EventReplayer.sol", "EventReplayer"},
	} {
		require.Equal(t, deployedCode(t, artifactsFS, expected.file, expected.contract),
			publicProjectionGenesis.Alloc[codeNamespace(expected.proxy)].Code,
			"embedded projection bytecode for %s", expected.contract)
	}
}
