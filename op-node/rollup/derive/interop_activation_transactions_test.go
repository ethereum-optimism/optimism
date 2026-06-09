package derive

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/stretchr/testify/require"
)

func TestInteropSetFeatureTx(t *testing.T) {
	encoded, err := interopSetFeatureTx()
	require.NoError(t, err)

	from, dep := toDepositTxn(t, encoded)
	require.Equal(t, L1InfoDepositerAddress, from)
	require.NotNil(t, dep.To())
	require.Equal(t, predeploys.L1BlockAddr, *dep.To())
	require.Equal(t, big.NewInt(0), dep.Value())
	require.Equal(t, big.NewInt(0), dep.Mint())
	require.Equal(t, interopSetFeatureGas, dep.Gas())

	expected := UpgradeDepositSource{Intent: "Interop pre: setFeature(INTEROP)"}
	require.Equal(t, expected.SourceHash(), dep.SourceHash())

	// Calldata: setFeature(bytes32) selector + "INTEROP" right-padded to 32 bytes.
	require.Len(t, dep.Data(), 4+32)
	var expectedFeature [32]byte
	copy(expectedFeature[:], "INTEROP")
	require.Equal(t, expectedFeature[:], dep.Data()[4:])
}

func TestInteropETHLiquidityFundingTx(t *testing.T) {
	encoded, err := interopETHLiquidityFundingTx()
	require.NoError(t, err)

	from, dep := toDepositTxn(t, encoded)
	require.Equal(t, L1InfoDepositerAddress, from)
	require.NotNil(t, dep.To())
	require.Equal(t, predeploys.ETHLiquidityAddr, *dep.To())
	require.Equal(t, InteropETHLiquidityFundingAmount(), dep.Mint())
	require.Equal(t, InteropETHLiquidityFundingAmount(), dep.Value())
	require.Equal(t, interopETHLiquidityFundGas, dep.Gas())

	expected := UpgradeDepositSource{Intent: "Interop post: ETHLiquidity Funding"}
	require.Equal(t, expected.SourceHash(), dep.SourceHash())
}

func TestInteropActivationUpgradeTransactions(t *testing.T) {
	bundleTxs, bundleGas, err := UpgradeTransactions(forks.Lagoon)
	require.NoError(t, err)

	singleChainTxs, singleChainGas, err := InteropActivationUpgradeTransactions(false)
	require.NoError(t, err)
	require.Equal(t, bundleTxs, singleChainTxs)
	require.Equal(t, bundleGas, singleChainGas)

	multiChainTxs, multiChainGas, err := InteropActivationUpgradeTransactions(true)
	require.NoError(t, err)
	require.Len(t, multiChainTxs, len(bundleTxs)+2)
	require.Equal(t, bundleTxs, multiChainTxs[1:len(multiChainTxs)-1])
	require.Equal(t, interopSetFeatureGas+bundleGas+interopETHLiquidityFundGas, multiChainGas)

	setFeatureTx, err := interopSetFeatureTx()
	require.NoError(t, err)
	require.Equal(t, setFeatureTx, multiChainTxs[0])

	fundingTx, err := interopETHLiquidityFundingTx()
	require.NoError(t, err)
	require.Equal(t, fundingTx, multiChainTxs[len(multiChainTxs)-1])
}

func TestUpgradeTransactionsInterop(t *testing.T) {
	txs, gas, err := UpgradeTransactions(forks.Lagoon)
	require.NoError(t, err)

	// 26 implementation deployments + L2CM deployment + upgradePredeploys = 28.
	require.Len(t, txs, 28)

	// First tx: StorageSetter implementation deployment (qualified intent).
	first := UpgradeDepositSource{Intent: "Interop 0: Deploy StorageSetter Implementation"}
	_, dep0 := toDepositTxn(t, txs[0])
	require.Equal(t, first.SourceHash(), dep0.SourceHash())

	// Last tx: L2ProxyAdmin upgradePredeploys.
	last := UpgradeDepositSource{Intent: "Interop 27: L2ProxyAdmin Upgrade Predeploys"}
	_, depLast := toDepositTxn(t, txs[len(txs)-1])
	require.Equal(t, last.SourceHash(), depLast.SourceHash())

	// Total gas equals sum of per-tx limits.
	var sumGas uint64
	for _, tx := range txs {
		_, dep := toDepositTxn(t, tx)
		sumGas += dep.Gas()
	}
	require.Equal(t, gas, sumGas)
}
