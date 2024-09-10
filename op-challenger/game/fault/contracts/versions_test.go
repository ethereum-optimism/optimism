package contracts

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var versionABI = `[{
    "inputs": [],
    "name": "version",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }]`

func TestVersionedBuilder(t *testing.T) {
	var builder VersionedBuilder[string]
	builder.AddVersion(1, 1, func() (string, error) { return "v1.1", nil })
	builder.AddVersion(1, 2, func() (string, error) { return "v1.2", nil })

	require.Equal(t, "v1.1", buildWithVersion(t, builder, "1.1.0"))
	require.Equal(t, "v1.1", buildWithVersion(t, builder, "1.1.1"))
	require.Equal(t, "v1.1", buildWithVersion(t, builder, "1.1.2"))
	require.Equal(t, "default", buildWithVersion(t, builder, "1.10.0"))
}

func buildWithVersion(t *testing.T, builder VersionedBuilder[string], version string) string {
	addr := common.Address{0xaa}
	contractABI := mustParseAbi(([]byte)(versionABI))
	stubRpc := batchingTest.NewAbiBasedRpc(t, addr, contractABI)
	stubRpc.SetResponse(addr, methodVersion, rpcblock.Latest, nil, []interface{}{version})
	actual, err := builder.Build(context.Background(), batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize), contractABI, addr, func() (string, error) {
		return "default", nil
	})
	require.NoError(t, err)
	return actual
}
