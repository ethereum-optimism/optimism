package derive

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestJovianNetworkTransactions(t *testing.T) {
	upgradeTxns, err := JovianNetworkUpgradeTransactions(true, true)
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 5)

	deployL1BlockSender, deployL1Block := toDepositTxn(t, upgradeTxns[0])
	require.Equal(t, deployL1BlockSender, common.HexToAddress("0x4210000000000000000000000000000000000006"))
	require.Equal(t, deployJovianL1BlockSource.SourceHash(), deployL1Block.SourceHash())
	require.Nil(t, deployL1Block.To())
	require.Equal(t, uint64(447_315), deployL1Block.Gas())
	require.Equal(t, l1BlockJovianDeploymentBytecode, deployL1Block.Data())

	updateL1BlockProxySender, updateL1BlockProxy := toDepositTxn(t, upgradeTxns[1])
	require.Equal(t, updateL1BlockProxySender, common.Address{})
	require.Equal(t, updateJovianL1BlockProxySource.SourceHash(), updateL1BlockProxy.SourceHash())
	require.NotNil(t, updateL1BlockProxy.To())
	require.Equal(t, *updateL1BlockProxy.To(), common.HexToAddress("0x4200000000000000000000000000000000000015"))
	require.Equal(t, uint64(50_000), updateL1BlockProxy.Gas())
	require.Equal(t, "0x3659cfe60000000000000000000000003ba4007f5c922fbb33c454b41ea7a1f11e83df2c", hexutil.Encode(updateL1BlockProxy.Data()))

	deployGasPriceOracleSender, deployGasPriceOracle := toDepositTxn(t, upgradeTxns[2])
	require.Equal(t, deployGasPriceOracleSender, common.HexToAddress("0x4210000000000000000000000000000000000007"))
	require.Equal(t, deployJovianGasPriceOracleSource.SourceHash(), deployGasPriceOracle.SourceHash())
	require.Nil(t, deployGasPriceOracle.To())
	require.Equal(t, uint64(1_625_000), deployGasPriceOracle.Gas())
	require.Equal(t, gasPriceOracleJovianDeploymentBytecode, deployGasPriceOracle.Data())

	updateGasPriceOracleSender, updateGasPriceOracle := toDepositTxn(t, upgradeTxns[3])
	require.Equal(t, updateGasPriceOracleSender, common.Address{})
	require.Equal(t, updateJovianGasPriceOracleSource.SourceHash(), updateGasPriceOracle.SourceHash())
	require.NotNil(t, updateGasPriceOracle.To())
	require.Equal(t, *updateGasPriceOracle.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(50_000), updateGasPriceOracle.Gas())
	require.Equal(t, "0x3659cfe60000000000000000000000004f1db3c6abd250ba86e0928471a8f7db3afd88f1", hexutil.Encode(updateGasPriceOracle.Data()))

	gpoSetJovianSender, gpoSetJovian := toDepositTxn(t, upgradeTxns[4])
	require.Equal(t, gpoSetJovianSender, common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"))
	require.Equal(t, enableJovianSource.SourceHash(), gpoSetJovian.SourceHash())
	require.NotNil(t, gpoSetJovian.To())
	require.Equal(t, *gpoSetJovian.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(90_000), gpoSetJovian.Gas())
	require.Equal(t, "0xb3d72079", hexutil.Encode(gpoSetJovian.Data()))
}
