package derive

import (
	"bytes"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestReadNUTBundle(t *testing.T) {
	f, err := os.Open("testdata/test-nut.json")
	require.NoError(t, err)
	defer f.Close()

	bundle, err := readNUTBundle("Test", f)
	require.NoError(t, err)

	require.Equal(t, forks.Name("Test"), bundle.ForkName)
	require.Equal(t, "1.0.0", bundle.Metadata.Version)
	require.Len(t, bundle.Transactions, 2)

	// First tx: no value field, zero address from
	tx0 := bundle.Transactions[0]
	require.Equal(t, "First Transaction", tx0.Intent)
	require.Equal(t, common.Address{}, tx0.From)
	require.NotNil(t, tx0.To)
	require.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), *tx0.To)
	require.Equal(t, common.FromHex("0xabcdef"), []byte(tx0.Data))
	require.Equal(t, uint64(1000000), tx0.GasLimit)

	// Second tx: non-zero from
	tx1 := bundle.Transactions[1]
	require.Equal(t, "Second Transaction", tx1.Intent)
	require.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000abba"), tx1.From)
	require.NotNil(t, tx1.To)
	require.Equal(t, uint64(5000000), tx1.GasLimit)
}

func TestNUTBundleToDepositTransactions(t *testing.T) {
	f, err := os.Open("testdata/test-nut.json")
	require.NoError(t, err)
	defer f.Close()

	bundle, err := readNUTBundle("Test", f)
	require.NoError(t, err)

	txs, err := bundle.toDepositTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 2)

	// Verify first tx: qualified intent is "Test 0: First Transaction"
	expectedSource0 := UpgradeDepositSource{Intent: "Test 0: First Transaction"}
	from0, dep0 := toDepositTxn(t, txs[0])
	require.Equal(t, common.Address{}, from0)
	require.Equal(t, expectedSource0.SourceHash(), dep0.SourceHash())
	require.NotNil(t, dep0.To())
	require.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), *dep0.To())
	require.Equal(t, uint64(1000000), dep0.Gas())
	require.Equal(t, common.FromHex("0xabcdef"), dep0.Data())
	require.Equal(t, big.NewInt(0), dep0.Value())

	// Verify second tx: qualified intent is "Test 1: Second Transaction"
	expectedSource1 := UpgradeDepositSource{Intent: "Test 1: Second Transaction"}
	from1, dep1 := toDepositTxn(t, txs[1])
	require.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000abba"), from1)
	require.Equal(t, expectedSource1.SourceHash(), dep1.SourceHash())
	require.Equal(t, uint64(5000000), dep1.Gas())
	require.Equal(t, big.NewInt(0), dep1.Value())
	// Source hashes must be unique
	require.NotEqual(t, dep0.SourceHash(), dep1.SourceHash())
}

func TestReadNUTBundleInvalidJSON(t *testing.T) {
	_, err := readNUTBundle("Test", bytes.NewReader([]byte(`{invalid`)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse NUT bundle")
}

func TestNUTBundleMissingIntent(t *testing.T) {
	jsonData := []byte(`{
		"metadata": {"version": "1.0.0"},
		"transactions": [{
			"from": "0x0000000000000000000000000000000000000000",
			"to": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			"data": "0xabcdef",
			"gasLimit": 1000000
		}]
	}`)

	bundle, err := readNUTBundle("Test", bytes.NewReader(jsonData))
	require.NoError(t, err)

	_, err = bundle.toDepositTransactions()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing intent")
}

func TestNUTBundleTotalGas(t *testing.T) {
	f, err := os.Open("testdata/test-nut.json")
	require.NoError(t, err)
	defer f.Close()

	bundle, err := readNUTBundle("Test", f)
	require.NoError(t, err)

	txs, err := bundle.toDepositTransactions()
	require.NoError(t, err)
	require.Len(t, txs, 2)
	require.Equal(t, uint64(1_000_000+5_000_000), bundle.totalGas())

	// Verify gas matches sum of individual deposit tx gas limits
	var sumGas uint64
	for _, tx := range txs {
		_, dep := toDepositTxn(t, tx)
		sumGas += dep.Gas()
	}
	require.Equal(t, bundle.totalGas(), sumGas)
}

func TestUpgradeTransactionsUnknownFork(t *testing.T) {
	_, _, err := UpgradeTransactions("UnknownFork")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no NUT bundle for fork")
}

// TestNUTBundleNullTo verifies that "to": null in JSON produces a contract creation (deploy) transaction.
// Although NUTs are expected to use Arachnid's deterministic deployer, this sending to null
// is how previous deployments have been handled and is useful to maintain going forward.
func TestNUTBundleNullTo(t *testing.T) {
	jsonData := []byte(`{
		"metadata": {"version": "1.0.0"},
		"transactions": [{
			"intent": "Deploy Contract",
			"from": "0x4210000000000000000000000000000000000006",
			"to": null,
			"data": "0xdeadbeef",
			"gasLimit": 500000
		}]
	}`)

	bundle, err := readNUTBundle("Test", bytes.NewReader(jsonData))
	require.NoError(t, err)
	require.Nil(t, bundle.Transactions[0].To)

	txs, err := bundle.toDepositTransactions()
	require.NoError(t, err)

	_, dep := toDepositTxn(t, txs[0])
	require.Nil(t, dep.To())
}
