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
	require.Equal(t, byte('I'), dep.Data()[4])
	require.Equal(t, byte('P'), dep.Data()[10])
	for _, b := range dep.Data()[11 : 4+32] {
		require.Equal(t, byte(0), b, "INTEROP must be right-padded with zeros")
	}
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

func TestUpgradeTransactionsInterop(t *testing.T) {
	txs, gas, err := UpgradeTransactions(forks.Interop)
	require.NoError(t, err)

	// 26 implementation deployments + L2CM deployment + upgradePredeploys = 28.
	require.Len(t, txs, 28)

	// First tx: StorageSetter implementation deployment (qualified intent).
	first := UpgradeDepositSource{Intent: "interop 0: Deploy StorageSetter Implementation"}
	_, dep0 := toDepositTxn(t, txs[0])
	require.Equal(t, first.SourceHash(), dep0.SourceHash())

	// Last tx: L2ProxyAdmin upgradePredeploys.
	last := UpgradeDepositSource{Intent: "interop 27: L2ProxyAdmin Upgrade Predeploys"}
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
