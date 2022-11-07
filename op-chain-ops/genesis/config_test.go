package genesis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/require"
)

func TestConfigMarshalUnmarshal(t *testing.T) {
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	decoded := new(DeployConfig)
	require.NoError(t, dec.Decode(decoded))
	encoded, err := json.MarshalIndent(decoded, "", "  ")

	require.NoError(t, err)
	require.JSONEq(t, string(b), string(encoded))
}

func TestUnmarshalL1StartingBlockTag(t *testing.T) {
	decoded := new(DeployConfig)
	require.NoError(t, json.Unmarshal([]byte(`{"l1StartingBlockTag": "earliest"}`), decoded))
	require.EqualValues(t, rpc.EarliestBlockNumber, *decoded.L1StartingBlockTag.BlockNumber)
	h := "0x86c7263d87140ca7cd9bf1bc9e95a435a7a0efc0ae2afaf64920c5b59a6393d4"
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`{"l1StartingBlockTag": "%s"}`, h)), decoded))
	require.EqualValues(t, common.HexToHash(h), *decoded.L1StartingBlockTag.BlockHash)
}
