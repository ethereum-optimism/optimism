package safe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestBatchJSONPrepareBedrock(t *testing.T) {
	testBatchJSON(t, "testdata/batch-prepare-bedrock.json")
}

func TestBatchJSONL2OO(t *testing.T) {
	testBatchJSON(t, "testdata/l2-output-oracle.json")
}

func testBatchJSON(t *testing.T, path string) {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	decoded := new(Batch)
	require.NoError(t, dec.Decode(decoded))
	data, err := json.Marshal(decoded)
	require.NoError(t, err)
	require.JSONEq(t, string(b), string(data))
}

func TestBatchAddCallFinalizeWithdrawalTransaction(t *testing.T) {
	file, err := os.ReadFile("testdata/portal-abi.json")
	require.NoError(t, err)
	portalABI, err := abi.JSON(bytes.NewReader(file))
	require.NoError(t, err)

	sig := "finalizeWithdrawalTransaction"
	argument := []any{
		bindings.TypesWithdrawalTransaction{
			Nonce:    big.NewInt(0),
			Sender:   common.Address{19: 0x01},
			Target:   common.Address{19: 0x02},
			Value:    big.NewInt(1),
			GasLimit: big.NewInt(2),
			Data:     []byte{},
		},
	}

	batch := new(Batch)
	to := common.Address{19: 0x01}
	value := big.NewInt(222)

	require.NoError(t, batch.AddCall(to, value, sig, argument, portalABI))
	d, err := json.MarshalIndent(batch, " ", "  ")
	require.NoError(t, err)
	fmt.Println(string(d))
}

func TestBatchAddCallDespostTransaction(t *testing.T) {
	file, err := os.ReadFile("testdata/portal-abi.json")
	require.NoError(t, err)
	portalABI, err := abi.JSON(bytes.NewReader(file))
	require.NoError(t, err)

	batch := new(Batch)
	to := common.Address{19: 0x01}
	value := big.NewInt(222)
	sig := "depositTransaction"
	argument := []any{
		common.Address{01},
		big.NewInt(2),
		uint64(100),
		false,
		[]byte{},
	}

	require.NoError(t, batch.AddCall(to, value, sig, argument, portalABI))
	d, err := json.MarshalIndent(batch, " ", "  ")
	require.NoError(t, err)
	fmt.Println(string(d))
}
