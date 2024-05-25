package safe

import (
	"bytes"
	"encoding/json"
	"errors"
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

// TestBatchAddCallFinalizeWithdrawalTransaction ensures that structs can be serialized correctly.
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

	require.NoError(t, batch.AddCall(to, value, sig, argument, &portalABI))
	require.NoError(t, batch.Check())
	require.Equal(t, batch.Transactions[0].Signature(), "finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes))")

	expected, err := os.ReadFile("testdata/finalize-withdrawal-tx.json")
	require.NoError(t, err)

	serialized, err := json.Marshal(batch)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(serialized))
}

// TestBatchAddCallDespostTransaction ensures that simple calls can be serialized correctly.
func TestBatchAddCallDespositTransaction(t *testing.T) {
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

	require.NoError(t, batch.AddCall(to, value, sig, argument, &portalABI))
	require.NoError(t, batch.Check())
	require.Equal(t, batch.Transactions[0].Signature(), "depositTransaction(address,uint256,uint64,bool,bytes)")

	expected, err := os.ReadFile("testdata/deposit-tx.json")
	require.NoError(t, err)

	serialized, err := json.Marshal(batch)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(serialized))
}

// TestBatchCheck checks for the various failure cases of Batch.Check
// as well as a simple check for a valid batch.
func TestBatchCheck(t *testing.T) {
	cases := []struct {
		name string
		bt   BatchTransaction
		err  error
	}{
		{
			name: "bad-input-count",
			bt: BatchTransaction{
				Method: ContractMethod{},
				InputValues: map[string]string{
					"foo": "bar",
				},
			},
			err: errors.New("expected 0 inputs but got 1"),
		},
		{
			name: "bad-calldata-too-small",
			bt: BatchTransaction{
				Data: []byte{0x01},
			},
			err: errors.New("must have at least 4 bytes of calldata, got 1"),
		},
		{
			name: "bad-calldata-mismatch",
			bt: BatchTransaction{
				Data: []byte{0x01, 0x02, 0x03, 0x04},
				Method: ContractMethod{
					Name: "foo",
				},
			},
			err: errors.New("data does not match signature"),
		},
		{
			name: "good-calldata",
			bt: BatchTransaction{
				Data: []byte{0xc2, 0x98, 0x55, 0x78},
				Method: ContractMethod{
					Name: "foo",
				},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.err, tc.bt.Check())
		})
	}
}
