package eth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckBlockHashRealPayloads recomputes the block hash of real L2 blocks, one per payload shape.
// The fixtures in testdata/payloads are the blocks served by op-reth, named <chain>-<number>, and
// converted to payloads with sources.RPCBlock.ExecutionPayloadEnvelope. Each row also changes one
// input of the header, which must change the hash.
func TestCheckBlockHashRealPayloads(t *testing.T) {
	for _, tc := range []struct {
		name   string
		file   string
		mutate func(env *ExecutionPayloadEnvelope)
	}{
		{
			name: "pre-Canyon: no withdrawals",
			file: "op-mainnet-110000007.json",
			mutate: func(env *ExecutionPayloadEnvelope) {
				tx := env.ExecutionPayload.Transactions[1]
				tx[len(tx)-1] ^= 1
			},
		},
		{
			name:   "Canyon: empty withdrawals",
			file:   "op-mainnet-115200001.json",
			mutate: func(env *ExecutionPayloadEnvelope) { env.ExecutionPayload.Withdrawals = nil },
		},
		{
			name:   "Ecotone: zero blob gas fields, parent beacon block root",
			file:   "op-sepolia-9000000.json",
			mutate: func(env *ExecutionPayloadEnvelope) { env.ParentBeaconBlockRoot[0] ^= 1 },
		},
		{
			name:   "Isthmus: withdrawals root is the message passer storage root, requests hash, Holocene extra data",
			file:   "op-sepolia-30000001.json",
			mutate: func(env *ExecutionPayloadEnvelope) { env.ExecutionPayload.WithdrawalsRoot[0] ^= 1 },
		},
		{
			name:   "Jovian: non-zero blob gas used is the DA footprint",
			file:   "op-sepolia-36000018.json",
			mutate: func(env *ExecutionPayloadEnvelope) { *env.ExecutionPayload.BlobGasUsed++ },
		},
		{
			name: "Karst: no header change, Jovian extra data",
			file: "op-sepolia-50000018.json",
			mutate: func(env *ExecutionPayloadEnvelope) {
				extra := env.ExecutionPayload.ExtraData
				extra[len(extra)-1] ^= 1
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "payloads", tc.file))
			require.NoError(t, err)
			var env ExecutionPayloadEnvelope
			require.NoError(t, json.Unmarshal(data, &env))

			actual, ok := env.CheckBlockHash()
			require.True(t, ok, "recomputed %s, payload has %s", actual, env.ExecutionPayload.BlockHash)

			tc.mutate(&env)
			actual, ok = env.CheckBlockHash()
			require.False(t, ok)
			require.NotEqual(t, env.ExecutionPayload.BlockHash, actual)
		})
	}
}
