package derive

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestJovianNetworkTransactions(t *testing.T) {
	upgradeTxns, err := JovianNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 3)

	deployGasPriceOracleSender, deployGasPriceOracle := toDepositTxn(t, upgradeTxns[0])
	require.Equal(t, deployGasPriceOracleSender, common.HexToAddress("0x4210000000000000000000000000000000000006"))
	require.Equal(t, deployJovianGasPriceOracleSource.SourceHash(), deployGasPriceOracle.SourceHash())
	require.Nil(t, deployGasPriceOracle.To())
	require.Equal(t, uint64(1_625_000), deployGasPriceOracle.Gas())
	require.Equal(t, gasPriceOracleJovianDeploymentBytecode, deployGasPriceOracle.Data())

	updateGasPriceOracleSender, updateGasPriceOracle := toDepositTxn(t, upgradeTxns[1])
	require.Equal(t, updateGasPriceOracleSender, common.Address{})
	require.Equal(t, updateJovianGasPriceOracleSource.SourceHash(), updateGasPriceOracle.SourceHash())
	require.NotNil(t, updateGasPriceOracle.To())
	require.Equal(t, *updateGasPriceOracle.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(50_000), updateGasPriceOracle.Gas())
	require.Equal(t, common.FromHex("3659cfe60000000000000000000000003ba4007f5c922fbb33c454b41ea7a1f11e83df2c"), updateGasPriceOracle.Data())

	gpoSetJovianSender, gpoSetJovian := toDepositTxn(t, upgradeTxns[2])
	require.Equal(t, gpoSetJovianSender, common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"))
	require.Equal(t, enableJovianSource.SourceHash(), gpoSetJovian.SourceHash())
	require.NotNil(t, gpoSetJovian.To())
	require.Equal(t, *gpoSetJovian.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(90_000), gpoSetJovian.Gas())
	require.Equal(t, common.FromHex("b3d72079"), gpoSetJovian.Data())
}
