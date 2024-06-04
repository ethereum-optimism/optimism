package derive

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSourcesMatchSpec(t *testing.T) {
	for _, test := range []struct {
		source       UpgradeDepositSource
		expectedHash string
	}{
		{
			source:       deployL1BlockSource,
			expectedHash: "0x877a6077205782ea15a6dc8699fa5ebcec5e0f4389f09cb8eda09488231346f8",
		},
		{
			source:       deployGasPriceOracleSource,
			expectedHash: "0xa312b4510adf943510f05fcc8f15f86995a5066bd83ce11384688ae20e6ecf42",
		},
		{
			source:       updateL1BlockProxySource,
			expectedHash: "0x18acb38c5ff1c238a7460ebc1b421fa49ec4874bdf1e0a530d234104e5e67dbc",
		},
		{
			source:       updateGasPriceOracleSource,
			expectedHash: "0xee4f9385eceef498af0be7ec5862229f426dec41c8d42397c7257a5117d9230a",
		},
		{
			source:       enableEcotoneSource,
			expectedHash: "0x0c1cb38e99dbc9cbfab3bb80863380b0905290b37eb3d6ab18dc01c1f3e75f93",
		},
		{
			source:       beaconRootsSource,
			expectedHash: "0x69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c",
		},
	} {
		require.Equal(t, common.HexToHash(test.expectedHash), test.source.SourceHash())
	}
}

func toDepositTxn(t *testing.T, data hexutil.Bytes) (common.Address, *types.Transaction) {
	txn := new(types.Transaction)
	err := txn.UnmarshalBinary(data)
	require.NoError(t, err)
	require.Truef(t, txn.IsDepositTx(), "expected deposit txn, got %v", txn.Type())
	require.False(t, txn.IsSystemTx())

	signer := types.NewLondonSigner(big.NewInt(420))
	from, err := signer.Sender(txn)
	require.NoError(t, err)

	return from, txn
}

func TestEcotoneNetworkTransactions(t *testing.T) {
	upgradeTxns, err := EcotoneNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 6)

	deployL1BlockSender, deployL1Block := toDepositTxn(t, upgradeTxns[0])
	require.Equal(t, deployL1BlockSender, common.HexToAddress("0x4210000000000000000000000000000000000000"))
	require.Equal(t, deployL1BlockSource.SourceHash(), deployL1Block.SourceHash())
	require.Nil(t, deployL1Block.To())
	require.Equal(t, uint64(375_000), deployL1Block.Gas())
	require.Equal(t, l1BlockDeploymentBytecode, deployL1Block.Data())

	deployGasPriceOracleSender, deployGasPriceOracle := toDepositTxn(t, upgradeTxns[1])
	require.Equal(t, deployGasPriceOracleSender, common.HexToAddress("0x4210000000000000000000000000000000000001"))
	require.Equal(t, deployGasPriceOracleSource.SourceHash(), deployGasPriceOracle.SourceHash())
	require.Nil(t, deployGasPriceOracle.To())
	require.Equal(t, uint64(1_000_000), deployGasPriceOracle.Gas())
	require.Equal(t, gasPriceOracleDeploymentBytecode, deployGasPriceOracle.Data())

	updateL1BlockProxySender, updateL1BlockProxy := toDepositTxn(t, upgradeTxns[2])
	require.Equal(t, updateL1BlockProxySender, common.Address{})
	require.Equal(t, updateL1BlockProxySource.SourceHash(), updateL1BlockProxy.SourceHash())
	require.NotNil(t, updateL1BlockProxy.To())
	require.Equal(t, *updateL1BlockProxy.To(), common.HexToAddress("0x4200000000000000000000000000000000000015"))
	require.Equal(t, uint64(50_000), updateL1BlockProxy.Gas())
	require.Equal(t, common.FromHex("0x3659cfe600000000000000000000000007dbe8500fc591d1852b76fee44d5a05e13097ff"), updateL1BlockProxy.Data())

	updateGasPriceOracleSender, updateGasPriceOracle := toDepositTxn(t, upgradeTxns[3])
	require.Equal(t, updateGasPriceOracleSender, common.Address{})
	require.Equal(t, updateGasPriceOracleSource.SourceHash(), updateGasPriceOracle.SourceHash())
	require.NotNil(t, updateGasPriceOracle.To())
	require.Equal(t, *updateGasPriceOracle.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(50_000), updateGasPriceOracle.Gas())
	require.Equal(t, common.FromHex("0x3659cfe6000000000000000000000000b528d11cc114e026f138fe568744c6d45ce6da7a"), updateGasPriceOracle.Data())

	gpoSetEcotoneSender, gpoSetEcotone := toDepositTxn(t, upgradeTxns[4])
	require.Equal(t, gpoSetEcotoneSender, common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"))
	require.Equal(t, enableEcotoneSource.SourceHash(), gpoSetEcotone.SourceHash())
	require.NotNil(t, gpoSetEcotone.To())
	require.Equal(t, *gpoSetEcotone.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	require.Equal(t, uint64(80_000), gpoSetEcotone.Gas())
	require.Equal(t, common.FromHex("0x22b90ab3"), gpoSetEcotone.Data())

	beaconRootsSender, beaconRoots := toDepositTxn(t, upgradeTxns[5])
	require.Equal(t, beaconRootsSender, common.HexToAddress("0x0B799C86a49DEeb90402691F1041aa3AF2d3C875"))
	require.Equal(t, beaconRootsSource.SourceHash(), beaconRoots.SourceHash())
	require.Nil(t, beaconRoots.To())
	require.Equal(t, uint64(250_000), beaconRoots.Gas())
	require.Equal(t, eip4788CreationData, beaconRoots.Data())
	require.NotEmpty(t, beaconRoots.Data())
}

func TestEip4788Params(t *testing.T) {
	require.Equal(t, EIP4788From, common.HexToAddress("0x0B799C86a49DEeb90402691F1041aa3AF2d3C875"))
	require.Equal(t, eip4788CreationData, common.FromHex("0x60618060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604d57602036146024575f5ffd5b5f35801560495762001fff810690815414603c575f5ffd5b62001fff01545f5260205ff35b5f5ffd5b62001fff42064281555f359062001fff015500"))
	require.NotEmpty(t, eip4788CreationData)
}
