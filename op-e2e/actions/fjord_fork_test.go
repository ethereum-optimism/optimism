package actions

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var (
	fjordGasPriceOracleCodeHash = common.HexToHash("0x9ceff82dc9f9bf592dc3954dde0ce8466864229caa8916fffc4005e2fde3d589")
	// https://basescan.org/tx/0x8debb2fe54200183fb8baa3c6dbd8e6ec2e4f7a4add87416cd60336b8326d16a
	txHex = "02f875822105819b8405709fb884057d460082e97f94273ca93a52b817294830ed7572aa591ccfa647fd80881249c58b0021fb3fc080a05bb08ccfd68f83392e446dac64d88a2d28e7072c06502dfabc4a77e77b5c7913a05878d53dd4ebba4f6367e572d524dffcabeec3abb1d8725ee3ac5dc32e1852e3"

	costTxSizeCoef int64 = -88_664
	costFastlzCoef int64 = 1_031_462
	costIntercept  int64 = -27_321_890
)

func TestFjordNetworkUpgradeTransactions(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	genesisBlock := hexutil.Uint64(0)
	fjordOffset := hexutil.Uint64(2)

	dp.DeployConfig.L1CancunTimeOffset = &genesisBlock // can be removed once Cancun on L1 is the default

	// Activate all forks at genesis, and schedule Fjord the block after
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisFjordTimeOffset = &fjordOffset

	require.NoError(t, dp.DeployConfig.Check(), "must have valid config")

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, _, _, sequencer, engine, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)
	ethCl := engine.EthClient()

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Get current implementations addresses (by slot) for L1Block + GasPriceOracle
	initialGasPriceOracleAddress, err := ethCl.StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, nil)
	require.NoError(t, err)

	// Get gas price from oracle
	gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, ethCl)
	require.NoError(t, err)

	// Build to the Fjord block
	sequencer.ActBuildL2ToFjord(t)

	// get latest block
	latestBlock, err := ethCl.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, sequencer.L2Unsafe().Number, latestBlock.Number().Uint64())

	transactions := latestBlock.Transactions()
	// L1Block: 1 set-L1-info + 1 deploys + 1 upgradeTo + 1 enable fjord on GPO
	// See [derive.FjordNetworkUpgradeTransactions]
	require.Equal(t, 4, len(transactions))

	// All transactions are successful
	for i := 1; i < 4; i++ {
		txn := transactions[i]
		receipt, err := ethCl.TransactionReceipt(context.Background(), txn.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		require.NotEmpty(t, txn.Data(), "upgrade tx must provide input data")
	}

	expectedGasPriceOracleAddress := crypto.CreateAddress(derive.GasPriceOracleFjordDeployerAddress, 0)

	// Gas Price Oracle Proxy is updated
	updatedGasPriceOracleAddress, err := ethCl.StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	require.Equal(t, expectedGasPriceOracleAddress, common.BytesToAddress(updatedGasPriceOracleAddress))
	require.NotEqualf(t, initialGasPriceOracleAddress, updatedGasPriceOracleAddress, "Gas Price Oracle Proxy address should have changed")
	verifyCodeHashMatches(t, ethCl, expectedGasPriceOracleAddress, fjordGasPriceOracleCodeHash)

	// Check that Fjord was activated
	isFjord, err := gasPriceOracle.IsFjord(nil)
	require.NoError(t, err)
	require.True(t, isFjord)

	// Check GetL1Fee is updated
	txData, err := hex.DecodeString(txHex)
	require.NoError(t, err)

	used, err := gasPriceOracle.GetL1Fee(&bind.CallOpts{}, txData)
	require.NoError(t, err)

	fastLzLength := types.FlzCompressLen(txData)

	cost := fjordL1Cost(t, gasPriceOracle, int64(fastLzLength), int64(len(txData)))

	require.Equal(t, cost.Uint64(), used.Uint64())

	// Check that L1FeeUppberBound works
	upperBound, err := gasPriceOracle.GetL1FeeUpperBound(&bind.CallOpts{}, big.NewInt(int64(len(txData))))
	require.NoError(t, err)

	flzUpperBound := len(txData) + len(txData)/255 + 16

	upperBoundCost := fjordL1Cost(t, gasPriceOracle, int64(flzUpperBound), int64(len(txData)))
	require.Equal(t, upperBoundCost.Uint64(), upperBound.Uint64())
}

func FuzzFastLz(f *testing.F) {
	f.Fuzz(func(gt *testing.T, data []byte) {
		if len(data) <= 71 {
			return
		}

		t := NewDefaultTesting(gt)
		dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
		genesisBlock := hexutil.Uint64(0)

		dp.DeployConfig.L1CancunTimeOffset = &genesisBlock // can be removed once Cancun on L1 is the default

		// Activate all forks at genesis, and schedule Fjord the block after
		dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisBlock

		require.NoError(t, dp.DeployConfig.Check(), "must have valid config")

		sd := e2eutils.Setup(t, dp, defaultAlloc)
		log := testlog.Logger(t, log.LvlDebug)
		_, _, _, sequencer, engine, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)
		ethCl := engine.EthClient()

		// start op-nodes
		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		// Build to the Fjord block
		sequencer.ActBuildL2ToFjord(t)

		// Get gas price from oracle
		gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, ethCl)
		require.NoError(t, err)

		used, err := gasPriceOracle.GetL1Fee(&bind.CallOpts{}, data)
		require.NoError(t, err)

		fastLzLength := types.FlzCompressLen(data)
		cost := fjordL1Cost(t, gasPriceOracle, int64(fastLzLength), int64(len(data)))

		require.Equal(t, cost.Uint64(), used.Uint64())
	})
}

// The new cost function:
// l1BaseFeeScaled = l1BaseFeeScalar * l1BaseFee * 16
// l1BlobFeeScaled = l1BlobFeeScalar * l1BlobBaseFee
// l1FeeScaled = l1BaseFeeScaled + l1BlobFeeScaled
// ((intercept + fastlzCoef*fastlzLength + uncompressedTxCoef*uncompressedTxSize) * l1FeeScaled) / 1e12
func fjordL1Cost(
	t require.TestingT,
	gasPriceOracle *bindings.GasPriceOracleCaller,
	fastLzLength,
	unsignedTxSize int64,
) *big.Int {
	baseFeeScalar, err := gasPriceOracle.BaseFeeScalar(nil)
	require.NoError(t, err)
	l1BaseFee, err := gasPriceOracle.L1BaseFee(nil)
	require.NoError(t, err)
	blobBaseFeeScalar, err := gasPriceOracle.BlobBaseFeeScalar(nil)
	require.NoError(t, err)
	blobBaseFee, err := gasPriceOracle.BlobBaseFee(nil)
	require.NoError(t, err)

	feeScaled := new(big.Int).Mul(new(big.Int).SetUint64(uint64(baseFeeScalar)), big.NewInt(16))
	feeScaled = new(big.Int).Mul(feeScaled, l1BaseFee)
	feeScaled = new(big.Int).Add(feeScaled, new(big.Int).Mul(new(big.Int).SetUint64(uint64(blobBaseFeeScalar)), blobBaseFee))

	cost := new(big.Int).Mul(new(big.Int).SetInt64(costFastlzCoef), new(big.Int).SetInt64(fastLzLength+68))
	cost = new(big.Int).Add(cost, new(big.Int).SetInt64(costIntercept))
	cost = new(big.Int).Add(cost, new(big.Int).Mul(new(big.Int).SetInt64(costTxSizeCoef), new(big.Int).SetInt64(unsignedTxSize+68)))
	require.True(t, cost.Sign() >= 0)

	cost = new(big.Int).Mul(cost, feeScaled)
	cost = new(big.Int).Div(cost, new(big.Int).SetInt64(int64(1e12)))

	return cost
}
