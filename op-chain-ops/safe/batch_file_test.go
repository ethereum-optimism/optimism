package safe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

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

func TestBatchAddCall(t *testing.T) {
	file, err := os.ReadFile("testdata/portal-abi.json")
	require.NoError(t, err)
	portalABI, err := abi.JSON(bytes.NewReader(file))
	require.NoError(t, err)

	batch := new(Batch)

	to := common.Address{19: 0x01}
	value := big.NewInt(222)
	/*
		sig := "depositTransaction"
		args := []any{
			common.Address{01},
			big.NewInt(2),
			uint64(100),
			false,
			[]byte{},
		}
	*/
	sig := "finalizeWithdrawalTransaction"
	args := []any{
		[]any{
			big.NewInt(0),
			common.Address{19: 0x01},
			common.Address{19: 0x02},
			big.NewInt(1),
			big.NewInt(2),
			[]byte{},
		},
	}

	require.NoError(t, batch.AddCall(to, value, sig, args, portalABI))

	d, err := json.MarshalIndent(batch, " ", "  ")
	require.NoError(t, err)
	fmt.Println(string(d))
}
