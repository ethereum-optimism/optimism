package derive

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.Equal(t, common.HexToHash(test.expectedHash), test.source.SourceHash())
	}
}

func toDepositTxn(t *testing.T, data hexutil.Bytes) (common.Address, *types.Transaction) {
	txn := new(types.Transaction)
	err := txn.UnmarshalBinary(data)
	require.NoError(t, err)
	require.Truef(t, txn.IsDepositTx(), "expected deposit txn, got %v", txn.Type())
	assert.False(t, txn.IsSystemTx())

	signer := types.NewLondonSigner(big.NewInt(420))
	from, err := signer.Sender(txn)
	require.NoError(t, err)

	return from, txn
}

func TestEcotoneNetworkTransactions(t *testing.T) {
	upgradeTxns, err := EcotoneNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 5)

	deployL1BlockSender, deployL1Block := toDepositTxn(t, upgradeTxns[0])
	assert.Equal(t, deployL1BlockSender, common.HexToAddress("0x4210000000000000000000000000000000000000"))
	assert.Equal(t, deployL1BlockSource.SourceHash(), deployL1Block.SourceHash())
	assert.Nil(t, deployL1Block.To())
	assert.Equal(t, uint64(300_000), deployL1Block.Gas())
	//TODO: assert bytecode when contracts PR is merged

	deployGasPriceOracleSender, deployGasPriceOracle := toDepositTxn(t, upgradeTxns[1])
	assert.Equal(t, deployGasPriceOracleSender, common.HexToAddress("0x4210000000000000000000000000000000000001"))
	assert.Equal(t, deployGasPriceOracleSource.SourceHash(), deployGasPriceOracle.SourceHash())
	assert.Nil(t, deployGasPriceOracle.To())
	assert.Equal(t, uint64(500_000), deployGasPriceOracle.Gas())
	//TODO: assert bytecode when contracts PR is merged

	updateL1BlockProxySender, updateL1BlockProxy := toDepositTxn(t, upgradeTxns[2])
	assert.Equal(t, updateL1BlockProxySender, common.Address{})
	assert.Equal(t, updateL1BlockProxySource.SourceHash(), updateL1BlockProxy.SourceHash())
	require.NotNil(t, updateL1BlockProxy.To())
	assert.Equal(t, *updateL1BlockProxy.To(), common.HexToAddress("0x4200000000000000000000000000000000000015"))
	assert.Equal(t, uint64(200_000), updateL1BlockProxy.Gas())
	assert.Equal(t, common.FromHex("0x3659cfe600000000000000000000000007dbe8500fc591d1852b76fee44d5a05e13097ff"), updateL1BlockProxy.Data())

	updateGasPriceOracleSender, updateGasPriceOracle := toDepositTxn(t, upgradeTxns[3])
	assert.Equal(t, updateGasPriceOracleSender, common.Address{})
	assert.Equal(t, updateGasPriceOracleSource.SourceHash(), updateGasPriceOracle.SourceHash())
	require.NotNil(t, updateGasPriceOracle.To())
	assert.Equal(t, *updateGasPriceOracle.To(), common.HexToAddress("0x420000000000000000000000000000000000000F"))
	assert.Equal(t, uint64(200_000), updateGasPriceOracle.Gas())
	assert.Equal(t, common.FromHex("0x3659cfe6000000000000000000000000b528d11cc114e026f138fe568744c6d45ce6da7a"), updateGasPriceOracle.Data())

	//TODO: verify setEcotone when contracts PR is merged

	beaconRootsSender, beaconRoots := toDepositTxn(t, upgradeTxns[4])
	assert.Equal(t, beaconRootsSender, common.HexToAddress("0x0B799C86a49DEeb90402691F1041aa3AF2d3C875"))
	assert.Equal(t, beaconRootsSource.SourceHash(), beaconRoots.SourceHash())
	assert.Nil(t, beaconRoots.To())
	assert.Equal(t, uint64(250_000), beaconRoots.Gas())
	assert.Equal(t, eip4788CreationData, beaconRoots.Data())
}

func TestEip4788Params(t *testing.T) {
	assert.Equal(t, eip4788From, common.HexToAddress("0x0B799C86a49DEeb90402691F1041aa3AF2d3C875"))
	assert.Equal(t, eip4788CreationData, common.Hex2Bytes("0x60618060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604d57602036146024575f5ffd5b5f35801560495762001fff810690815414603c575f5ffd5b62001fff01545f5260205ff35b5f5ffd5b62001fff42064281555f359062001fff015500"))
}
