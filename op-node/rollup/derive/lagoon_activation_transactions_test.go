package derive

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestInteropSetFeatureDeposit(t *testing.T) {
	encoded, err := interopSetFeatureDeposit().MarshalBinary()
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

func TestInteropETHLiquidityFundingDeposit(t *testing.T) {
	encoded, err := interopETHLiquidityFundingDeposit().MarshalBinary()
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

func TestLagoonActivationUpgradeTransactions(t *testing.T) {
	bundleTxs, gotGas, err := UpgradeTransactions(forks.Lagoon)
	require.NoError(t, err)

	// Derive the expectation from the deposits themselves rather than from the getters under test,
	// so this cannot pass by agreeing with a wrong implementation.
	var bundleTxGas uint64
	for _, tx := range bundleTxs {
		_, dep := toDepositTxn(t, tx)
		bundleTxGas += dep.Gas()
	}
	wantGas := bundleTxGas + interopSetFeatureGas + interopETHLiquidityFundGas
	require.Equal(t, wantGas, gotGas, "UpgradeTransactions must report the full activation reservation")

	// Only the tx set varies with the dependency set; the reserved gas covers the wrappers either
	// way, so the reconstruction can subtract it without knowing the dependency set.
	singleChainTxs, singleChainGas, err := LagoonActivationUpgradeTransactions(false)
	require.NoError(t, err)
	require.Equal(t, bundleTxs, singleChainTxs)
	require.Equal(t, wantGas, singleChainGas)

	multiChainTxs, multiChainGas, err := LagoonActivationUpgradeTransactions(true)
	require.NoError(t, err)
	require.Len(t, multiChainTxs, len(bundleTxs)+2)
	require.Equal(t, bundleTxs, multiChainTxs[1:len(multiChainTxs)-1])
	require.Equal(t, wantGas, multiChainGas)

	// Only the multi-chain branch's deposits actually sum to the reservation.
	var multiChainTxGas uint64
	for _, tx := range multiChainTxs {
		_, dep := toDepositTxn(t, tx)
		multiChainTxGas += dep.Gas()
	}
	require.Equal(t, wantGas, multiChainTxGas)

	setFeatureTx, err := interopSetFeatureDeposit().MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(setFeatureTx), multiChainTxs[0])

	fundingTx, err := interopETHLiquidityFundingDeposit().MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(fundingTx), multiChainTxs[len(multiChainTxs)-1])
}

// TestLagoonUpgradeGasAddEqualsStrip pins the invariant that the gas added to the Lagoon activation
// block's gas limit equals the gas the system-config reconstruction subtracts again for the next
// block. The reconstruction sees only the rollup config and a timestamp — never the dependency set
// — so the added amount must be identical in both the single-chain and multi-chain branches.
func TestLagoonUpgradeGasAddEqualsStrip(t *testing.T) {
	const blockTime = uint64(2)
	lagoonTime := uint64(1000)

	cfg := &rollup.Config{BlockTime: blockTime}
	cfg.ActivateAtGenesis(forks.Karst)
	cfg.SetActivationTime(forks.Lagoon, &lagoonTime)
	require.True(t, cfg.IsActivationBlockForFork(lagoonTime, forks.Lagoon))

	stripped := upgradeGasToStrip(cfg, lagoonTime)
	require.NotZero(t, stripped, "Lagoon must strip a non-zero amount")

	for _, activateInteropContracts := range []bool{false, true} {
		_, added, err := LagoonActivationUpgradeTransactions(activateInteropContracts)
		require.NoError(t, err)
		require.Equalf(t, added, stripped,
			"activateInteropContracts=%v: gas added to the activation block must equal the gas stripped for the next block",
			activateInteropContracts)
	}
}

func TestUpgradeTransactionsInterop(t *testing.T) {
	txs, gas, err := UpgradeTransactions(forks.Lagoon)
	require.NoError(t, err)

	// 27 implementation deployments + L2CM deployment + upgradePredeploys = 29.
	require.Len(t, txs, 29)

	// First tx: StorageSetter implementation deployment (qualified intent).
	first := UpgradeDepositSource{Intent: "Interop 0: Deploy StorageSetter Implementation"}
	_, dep0 := toDepositTxn(t, txs[0])
	require.Equal(t, first.SourceHash(), dep0.SourceHash())

	// Last tx: L2ProxyAdmin upgradePredeploys.
	last := UpgradeDepositSource{Intent: "Interop 28: L2ProxyAdmin Upgrade Predeploys"}
	_, depLast := toDepositTxn(t, txs[len(txs)-1])
	require.Equal(t, last.SourceHash(), depLast.SourceHash())

	// The reserved gas deliberately exceeds these transactions' own limits: it also covers the two
	// wrapper deposits that only a multi-chain activation emits. See UpgradeGas.
	var sumGas uint64
	for _, tx := range txs {
		_, dep := toDepositTxn(t, tx)
		sumGas += dep.Gas()
	}
	require.Equal(t, sumGas+interopSetFeatureGas+interopETHLiquidityFundGas, gas)
}

// TestLagoonNUTBundleL2ToL2MessengerExpiryPeriod checks that the embedded Lagoon NUT bundle deploys
// the L2ToL2CrossDomainMessenger with the production expiry period: the 7-day interop message
// expiry window plus a day of margin.
func TestLagoonNUTBundleL2ToL2MessengerExpiryPeriod(t *testing.T) {
	bundle, err := nutBundleForFork(forks.Lagoon)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(8*24*60*60), l2ToL2MessengerExpiryPeriod(t, bundle))
}

// l2ToL2MessengerExpiryPeriod returns the expiry period a NUT bundle deploys the
// L2ToL2CrossDomainMessenger implementation with. The deployment calls
// ConditionalDeployer.deploy(bytes32 salt, bytes code), and the code ends with the constructor's
// one argument.
func l2ToL2MessengerExpiryPeriod(t *testing.T, bundle *nutBundle) *big.Int {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	bytesType, err := abi.NewType("bytes", "", nil)
	require.NoError(t, err)
	deployArgs := abi.Arguments{{Type: bytes32Type}, {Type: bytesType}}

	for _, tx := range bundle.Transactions {
		if tx.Intent != "Deploy L2ToL2CrossDomainMessenger Implementation" {
			continue
		}
		require.Greater(t, len(tx.Data), 4, "the deployment must carry calldata")
		values, err := deployArgs.Unpack(tx.Data[4:])
		require.NoError(t, err)
		code := values[1].([]byte)
		require.GreaterOrEqual(t, len(code), 32, "the deployment code must end with the constructor argument")
		return new(big.Int).SetBytes(code[len(code)-32:])
	}
	t.Fatal("the bundle does not deploy the L2ToL2CrossDomainMessenger implementation")
	return nil
}
