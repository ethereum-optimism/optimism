package safe

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchFileJSONPrepareBedrock(t *testing.T) {
	testBatchFileJSON(t, "testdata/batch-prepare-bedrock.json")
}

func TestBatchFileJSONL2OO(t *testing.T) {
	testBatchFileJSON(t, "testdata/l2-output-oracle.json")
}

func testBatchFileJSON(t *testing.T, path string) {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	decoded := new(BatchFile)
	require.NoError(t, dec.Decode(decoded))
	data, err := json.Marshal(decoded)
	require.NoError(t, err)
	require.JSONEq(t, string(b), string(data))
}
