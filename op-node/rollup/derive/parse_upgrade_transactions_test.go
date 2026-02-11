package derive

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestParseNUTBundle(t *testing.T) {
	data, err := os.ReadFile("testdata/test-nut.json")
	require.NoError(t, err)

	bundle, err := ParseNUTBundle(data)
	require.NoError(t, err)

	require.Equal(t, "1.0.0", bundle.Metadata.Version)
	require.Len(t, bundle.Transactions, 2)

	// First tx: no value field, zero address from
	tx0 := bundle.Transactions[0]
	require.Equal(t, common.Address{}, tx0.From)
	require.NotNil(t, tx0.To)
	require.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), *tx0.To)
	require.Equal(t, common.FromHex("0xabcdef"), []byte(tx0.Data))
	require.Equal(t, uint64(1000000), tx0.GasLimit)
	require.Nil(t, tx0.Value)

	// Second tx: has value and non-zero from
	tx1 := bundle.Transactions[1]
	require.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000abba"), tx1.From)
	require.NotNil(t, tx1.To)
	require.Equal(t, uint64(5000000), tx1.GasLimit)
	require.Equal(t, big.NewInt(100), tx1.Value)
}

func TestNUTBundleToDepositTransactions(t *testing.T) {
	data, err := os.ReadFile("testdata/test-nut.json")
	require.NoError(t, err)

	bundle, err := ParseNUTBundle(data)
	require.NoError(t, err)

	txs, err := bundle.ToDepositTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 2)

	// Verify first tx round-trips to a valid deposit tx
	from0, dep0 := toDepositTxn(t, txs[0])
	require.Equal(t, common.Address{}, from0)
	require.NotNil(t, dep0.To())
	require.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), *dep0.To())
	require.Equal(t, uint64(1000000), dep0.Gas())
	require.Equal(t, common.FromHex("0xabcdef"), dep0.Data())
	require.Equal(t, big.NewInt(0), dep0.Value())

	// Verify second tx round-trips with value
	from1, dep1 := toDepositTxn(t, txs[1])
	require.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000abba"), from1)
	require.Equal(t, uint64(5000000), dep1.Gas())
	require.Equal(t, big.NewInt(100), dep1.Value())
}

func TestParseNUTBundleInvalidJSON(t *testing.T) {
	_, err := ParseNUTBundle([]byte(`{invalid`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse NUT bundle")
}

// TestNUTBundleNullTo verifies that "to": null in JSON produces a contract creation (deploy) transaction.
// Although NUTs are expected to use Arachnid's deterministic deployer, this sending to null
// is how previous deployments have been handled and is useful to maintain going forward.
func TestNUTBundleNullTo(t *testing.T) {
	jsonData := []byte(`{
		"metadata": {"version": "1.0.0"},
		"transactions": [{
			"from": "0x4210000000000000000000000000000000000006",
			"to": null,
			"data": "0xdeadbeef",
			"gasLimit": 500000
		}]
	}`)

	bundle, err := ParseNUTBundle(jsonData)
	require.NoError(t, err)
	require.Nil(t, bundle.Transactions[0].To)

	txs, err := bundle.ToDepositTransactions()
	require.NoError(t, err)

	_, dep := toDepositTxn(t, txs[0])
	require.Nil(t, dep.To())
}
