package mon

import (
	"context"
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

func TestStatusLoader_GetStatus(t *testing.T) {
	tests := []struct {
		name   string
		status types.GameStatus
	}{
		{
			name:   "statusInProgress",
			status: types.GameStatusInProgress,
		},
		{
			name:   "statusDefenderWon",
			status: types.GameStatusDefenderWon,
		},
		{
			name:   "statusChallengerWon",
			status: types.GameStatusChallengerWon,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			stubRpc, caller := setupStatusLoaderTest(t)
			metrics := &mockCacheMetrics{}
			loader := NewStatusLoader(metrics, caller)
			stubRpc.SetResponse(fdgAddr, "status", batching.BlockLatest, nil, []interface{}{test.status})
			status, err := loader.GetStatus(context.Background(), fdgAddr)
			require.NoError(t, err)
			require.Equal(t, test.status, status)
		})
	}
}

func setupStatusLoaderTest(t *testing.T) (*batchingTest.AbiBasedRpc, *batching.MultiCaller) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	return stubRpc, caller
}

type mockCacheMetrics struct{}

func (m *mockCacheMetrics) CacheAdd(_ string, _ int, _ bool) {}
func (m *mockCacheMetrics) CacheGet(_ string, _ bool)        {}
