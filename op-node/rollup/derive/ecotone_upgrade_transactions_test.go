package derive

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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

func toDepositTxn(t *testing.T, data hexutil.Bytes) *types.Transaction {
	txn := new(types.Transaction)
	err := txn.UnmarshalBinary(data)
	require.NoError(t, err)
	require.Truef(t, txn.IsDepositTx(), "expected deposit txn, got %v", txn.Type())
	return txn
}

func TestEcotoneNetworkTransactions(t *testing.T) {
	upgradeTxns, err := EcotoneNetworkUpgradeTransactions()
	require.NoError(t, err)
	require.Len(t, upgradeTxns, 5)

	toDepositTxn(t, upgradeTxns[0])

}
