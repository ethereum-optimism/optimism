package extract

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/stretchr/testify/require"
)

var (
	fdgAddr = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
)

func TestMetadataCreator_CreateContract(t *testing.T) {
	tests := []struct {
		name        string
		game        types.GameMetadata
		expectedErr error
	}{
		{
			name: "validCannonGameType",
			game: types.GameMetadata{GameType: uint32(types.CannonGameType), Proxy: fdgAddr},
		},
		{
			name: "validPermissionedGameType",
			game: types.GameMetadata{GameType: uint32(types.PermissionedGameType), Proxy: fdgAddr},
		},
		{
			name: "validCannonKonaGameType",
			game: types.GameMetadata{GameType: uint32(types.CannonKonaGameType), Proxy: fdgAddr},
		},
		{
			name: "validAlphabetGameType",
			game: types.GameMetadata{GameType: uint32(types.AlphabetGameType), Proxy: fdgAddr},
		},
		{
			name: "validFastGameType",
			game: types.GameMetadata{GameType: uint32(types.FastGameType), Proxy: fdgAddr},
		},
		{
			name: "validSuperPermissionedGameType",
			game: types.GameMetadata{GameType: uint32(types.SuperPermissionedGameType), Proxy: fdgAddr},
		},
		{
			name: "validSuperCannonKonaGameType",
			game: types.GameMetadata{GameType: uint32(types.SuperCannonKonaGameType), Proxy: fdgAddr},
		},
		{
			name:        "InvalidGameType",
			game:        types.GameMetadata{GameType: 6, Proxy: fdgAddr},
			expectedErr: fmt.Errorf("unsupported game type: 6"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			caller, metrics := setupMetadataLoaderTest(t, test.game.GameType)
			creator := NewGameCallerCreator(metrics, caller)
			_, err := creator.CreateContract(context.Background(), test.game)
			require.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				require.Equal(t, 1, metrics.cacheAddCalls)
				require.Equal(t, 1, metrics.cacheGetCalls)
			}
			_, err = creator.CreateContract(context.Background(), test.game)
			require.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				require.Equal(t, 1, metrics.cacheAddCalls)
				require.Equal(t, 2, metrics.cacheGetCalls)
			}
		})
	}
}

func TestSuperPermissionedGameCaller_GetExtendedMetadata(t *testing.T) {
	block := rpcblock.ByNumber(123)
	expectedL1Head := common.Hash{0x11}
	expectedL2SequenceNumber := uint64(456)
	expectedRootClaim := common.Hash{0x22}
	expectedStatus := types.GameStatusChallengerWon

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, snapshots.LoadSuperPermissionedDisputeGameABI())
	stubRpc.SetResponse(fdgAddr, "l1Head", block, nil, []interface{}{expectedL1Head})
	stubRpc.SetResponse(fdgAddr, "l2SequenceNumber", block, nil, []interface{}{new(big.Int).SetUint64(expectedL2SequenceNumber)})
	stubRpc.SetResponse(fdgAddr, "rootClaim", block, nil, []interface{}{expectedRootClaim})
	stubRpc.SetResponse(fdgAddr, "status", block, nil, []interface{}{expectedStatus})

	caller := NewSuperPermissionedGameCaller(&mockCacheMetrics{}, fdgAddr, batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize))
	actual, err := caller.GetExtendedMetadata(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, expectedL1Head, actual.L1Head)
	require.Equal(t, expectedL2SequenceNumber, actual.L2SequenceNum)
	require.Equal(t, expectedRootClaim, actual.RootClaim)
	require.Equal(t, expectedStatus, actual.Status)
	require.Zero(t, actual.MaxClockDuration)

	claims, err := caller.GetAllClaims(context.Background(), block)
	require.NoError(t, err)
	require.Empty(t, claims)

	credits, err := caller.GetCredits(context.Background(), block, common.Address{0xaa}, common.Address{0xbb})
	require.NoError(t, err)
	require.Len(t, credits, 2)
	require.Equal(t, big.NewInt(0), credits[0])
	require.Equal(t, big.NewInt(0), credits[1])
}

func setupMetadataLoaderTest(t *testing.T, gameType uint32) (*batching.MultiCaller, *mockCacheMetrics) {
	fdgAbi := snapshots.LoadFaultDisputeGameABI()
	if gameType == uint32(types.SuperPermissionedGameType) {
		fdgAbi = snapshots.LoadSuperPermissionedDisputeGameABI()
	} else if gameType == uint32(types.SuperCannonKonaGameType) {
		fdgAbi = snapshots.LoadSuperFaultDisputeGameABI()
	}
	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	stubRpc.SetResponse(fdgAddr, "version", rpcblock.Latest, nil, []interface{}{"0.18.0"})
	stubRpc.SetResponse(fdgAddr, "gameType", rpcblock.Latest, nil, []interface{}{gameType})
	return caller, &mockCacheMetrics{}
}

type mockCacheMetrics struct {
	cacheAddCalls int
	cacheGetCalls int
	*contractMetrics.NoopMetrics
}

func (m *mockCacheMetrics) CacheAdd(_ string, _ int, _ bool) {
	m.cacheAddCalls++
}
func (m *mockCacheMetrics) CacheGet(_ string, _ bool) {
	m.cacheGetCalls++
}
