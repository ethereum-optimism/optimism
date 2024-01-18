package actions

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var (
	l1BlockCodeHash        = common.FromHex("0xc88a313aa75dc4fbf0b6850d9f9ae41e04243b7008cf3eadb29256d4a71c1dfd")
	gasPriceOracleCodeHash = common.FromHex("0x8b71360ea773b4cfaf1ae6d2bd15464a4e1e2e360f786e475f63aeaed8da0ae5")
)

func verifyCodeHashMatches(t *testing.T, client *ethclient.Client, address common.Address, expectedCodeHash []byte) {
	code, err := client.CodeAt(context.Background(), address, nil)
	require.NoError(t, err)
	codeHash := crypto.Keccak256Hash(code)
	require.Equal(t, expectedCodeHash, codeHash.Bytes())
}

func TestEcotoneNetworkUpgradeTransactions(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	genesisBlock := hexutil.Uint64(0)
	ecotoneOffset := hexutil.Uint64(2)

	dp.DeployConfig.L1CancunTimeOffset = &genesisBlock // can be removed once Cancun on L1 is the default

	// Activate all forks at genesis, and schedule Ecotone the block after
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &ecotoneOffset
	require.NoError(t, dp.DeployConfig.Check(), "must have valid config")

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, _, miner, sequencer, engine, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Get gas price from oracle
	gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, engine.EthClient())
	require.NoError(t, err)

	scalar, err := gasPriceOracle.Scalar(nil)
	require.NoError(t, err)
	require.True(t, scalar.Cmp(big.NewInt(0)) > 0, "scalar must start non-zero")
	require.True(t, scalar.Cmp(new(big.Int).SetUint64(dp.DeployConfig.GasPriceOracleScalar)) == 0, "must match deploy config")

	// Get current implementations addresses (by slot) for L1Block + GasPriceOracle
	initialGasPriceOracleAddress, err := engine.EthClient().StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, nil)
	require.NoError(t, err)
	initialL1BlockAddress, err := engine.EthClient().StorageAt(context.Background(), predeploys.L1BlockAddr, genesis.ImplementationSlot, nil)
	require.NoError(t, err)

	// Build to the ecotone block
	sequencer.ActBuildL2ToEcotone(t)

	// get latest block
	latestBlock, err := engine.EthClient().BlockByNumber(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, sequencer.L2Unsafe().Number, latestBlock.Number().Uint64())

	transactions := latestBlock.Transactions()
	// L1Block: 1 set-L1-info + 2 deploys + 2 upgradeTo + 1 enable ecotone on GPO + 1 4788 deploy
	// See [derive.EcotoneNetworkUpgradeTransactions]
	require.Equal(t, 7, len(transactions))

	l1Info, err := derive.L1BlockInfoFromBytes(sd.RollupCfg, latestBlock.Time(), transactions[0].Data())
	require.NoError(t, err)
	require.Equal(t, derive.L1InfoBedrockLen, len(transactions[0].Data()))
	require.Nil(t, l1Info.BlobBaseFee)

	// All transactions are successful
	for i := 1; i < 7; i++ {
		txn := transactions[i]
		receipt, err := engine.EthClient().TransactionReceipt(context.Background(), txn.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}

	expectedL1BlockAddress := crypto.CreateAddress(derive.L1BlockDeployerAddress, 0)
	expectedGasPriceOracleAddress := crypto.CreateAddress(derive.GasPriceOracleDeployerAddress, 0)

	// Gas Price Oracle Proxy is updated
	updatedGasPriceOracleAddress, err := engine.EthClient().StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	assert.Equal(t, expectedGasPriceOracleAddress, common.BytesToAddress(updatedGasPriceOracleAddress))
	assert.NotEqualf(t, initialGasPriceOracleAddress, updatedGasPriceOracleAddress, "Gas Price Oracle Proxy address should have changed")
	verifyCodeHashMatches(gt, engine.EthClient(), expectedGasPriceOracleAddress, gasPriceOracleCodeHash)

	// L1Block Proxy is updated
	updatedL1BlockAddress, err := engine.EthClient().StorageAt(context.Background(), predeploys.L1BlockAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	assert.Equal(t, expectedL1BlockAddress, common.BytesToAddress(updatedL1BlockAddress))
	assert.NotEqualf(t, initialL1BlockAddress, updatedL1BlockAddress, "L1Block Proxy address should have changed")
	verifyCodeHashMatches(gt, engine.EthClient(), expectedL1BlockAddress, l1BlockCodeHash)

	_, err = gasPriceOracle.Scalar(nil)
	require.ErrorContains(t, err, "scalar() is deprecated")

	cost, err := gasPriceOracle.GetL1Fee(nil, []byte{0, 1, 2, 3, 4})
	require.NoError(t, err)
	// Pre-ecotone the GPO getL1Fee contract erroneously returned the pre-regolith L1 fee.
	// Thus we do not assert the exact value here.
	require.Equal(t, cost.Uint64(), uint64(0), "expecting zero scalars within activation block")

	// Check that Ecotone was activated
	isEcotone, err := gasPriceOracle.IsEcotone(nil)
	require.NoError(t, err)
	require.True(t, isEcotone)

	// 4788 contract is deployed
	expected4788Address := crypto.CreateAddress(derive.EIP4788From, 0)
	code, err := engine.EthClient().CodeAt(context.Background(), expected4788Address, latestBlock.Number())
	require.NoError(t, err)
	require.Equal(t, code, predeploys.EIP4788ContractCode)

	// Build empty L2 block, to pass ecotone activation
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)

	// assert again, now that we are past activation
	_, err = gasPriceOracle.Scalar(nil)
	require.ErrorContains(t, err, "scalar() is deprecated")

	// test if the migrated scalar matches the deploy config
	basefeeScalar, err := gasPriceOracle.BaseFeeScalar(nil)
	require.NoError(t, err)
	require.True(t, uint64(basefeeScalar) == dp.DeployConfig.GasPriceOracleScalar, "must match deploy config")

	cost, err = gasPriceOracle.GetL1Fee(nil, []byte{0, 1, 2, 3, 4})
	require.NoError(t, err)
	require.Greater(t, cost.Uint64(), uint64(0), "expecting non-zero scalars after activation block")

	// Get L1Block info
	l1Block, err := bindings.NewL1BlockCaller(predeploys.L1BlockAddr, engine.EthClient())
	require.NoError(t, err)
	l1BlockInfo, err := l1Block.Timestamp(nil)
	require.NoError(t, err)
	assert.Greater(t, l1BlockInfo, uint64(0))

	l1OriginBlock, err := miner.EthClient().BlockByHash(context.Background(), sequencer.L2Unsafe().L1Origin.Hash)
	require.NoError(t, err)
	l1Basefee, err := l1Block.Basefee(nil)
	require.NoError(t, err)
	require.Equal(t, l1OriginBlock.BaseFee().Uint64(), l1Basefee.Uint64(), "basefee must match")

	// calldataGas*(l1BaseFee*16*l1BaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar)/16e6
	// _getCalldataGas in GPO adds the cost of 68 non-zero bytes for signature/rlp overhead.
	calldataGas := big.NewInt(4*16 + 1*4 + 68*16)
	expectedL1Fee := new(big.Int).Mul(calldataGas, l1Basefee)
	expectedL1Fee = expectedL1Fee.Mul(expectedL1Fee, big.NewInt(16))
	expectedL1Fee = expectedL1Fee.Mul(expectedL1Fee, new(big.Int).SetUint64(uint64(basefeeScalar)))
	expectedL1Fee = expectedL1Fee.Div(expectedL1Fee, big.NewInt(16e6))
	require.Equal(t, expectedL1Fee, cost, "expecting cost based on regular base fee scalar alone")
}
