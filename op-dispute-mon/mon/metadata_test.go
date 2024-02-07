package mon

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/stretchr/testify/require"
)

var (
	fdgAddr = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
)

func TestMetadataLoader_GetMetadata(t *testing.T) {
	tests := []struct {
		name      string
		status    types.GameStatus
		blockNum  uint64
		rootClaim common.Hash
	}{
		{
			name:      "statusInProgress",
			status:    types.GameStatusInProgress,
			blockNum:  1,
			rootClaim: common.HexToHash("0x1"),
		},
		{
			name:      "statusDefenderWon",
			status:    types.GameStatusDefenderWon,
			blockNum:  2,
			rootClaim: common.HexToHash("0x2"),
		},
		{
			name:      "statusChallengerWon",
			status:    types.GameStatusChallengerWon,
			blockNum:  3,
			rootClaim: common.HexToHash("0x3"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			stubRpc, caller := setupMetadataLoaderTest(t)
			metrics := &mockCacheMetrics{}
			loader := NewMetadataLoader(metrics, caller)
			stubRpc.SetResponse(fdgAddr, "status", batching.BlockLatest, nil, []interface{}{test.status})
			stubRpc.SetResponse(fdgAddr, "l2BlockNumber", batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(test.blockNum)})
			stubRpc.SetResponse(fdgAddr, "rootClaim", batching.BlockLatest, nil, []interface{}{test.rootClaim})
			blockNum, rootClaim, status, err := loader.GetGameMetadata(context.Background(), fdgAddr)
			require.NoError(t, err)
			require.Equal(t, test.status, status)
			require.Equal(t, test.blockNum, blockNum)
			require.Equal(t, test.rootClaim, rootClaim)
		})
	}
}

func setupMetadataLoaderTest(t *testing.T) (*batchingTest.AbiBasedRpc, *batching.MultiCaller) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	return stubRpc, caller
}

type mockCacheMetrics struct{}

func (m *mockCacheMetrics) CacheAdd(_ string, _ int, _ bool) {}
func (m *mockCacheMetrics) CacheGet(_ string, _ bool)        {}
