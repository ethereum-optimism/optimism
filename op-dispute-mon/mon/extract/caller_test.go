package extract

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
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

func setupMetadataLoaderTest(t *testing.T, gameType uint32) (*batching.MultiCaller, *mockCacheMetrics) {
	fdgAbi := snapshots.LoadFaultDisputeGameABI()
	if gameType == uint32(types.SuperPermissionedGameType) {
		fdgAbi = snapshots.LoadSuperPermissionedDisputeGameABI()
	} else if gameType == uint32(types.SuperCannonKonaGameType) {
		fdgAbi = snapshots.LoadSuperFaultDisputeGameABI()
	}
	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	if gameType != uint32(types.SuperPermissionedGameType) {
		stubRpc.SetResponse(fdgAddr, "version", rpcblock.Latest, nil, []interface{}{"0.18.0"})
		stubRpc.SetResponse(fdgAddr, "gameType", rpcblock.Latest, nil, []interface{}{gameType})
	}
	return caller, &mockCacheMetrics{}
}

func TestSuperPermissionedGameCaller(t *testing.T) {
	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, snapshots.LoadSuperPermissionedDisputeGameABI())
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	metrics := &mockCacheMetrics{}
	game := NewSuperPermissionedGameCaller(metrics, fdgAddr, caller)

	l1Head := common.Hash{0xaa}
	l2SequenceNumber := big.NewInt(1234)
	rootClaim := common.Hash{0xbb}
	status := types.GameStatusDefenderWon
	stubRpc.SetResponse(fdgAddr, "l1Head", rpcblock.Latest, nil, []interface{}{l1Head})
	stubRpc.SetResponse(fdgAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{l2SequenceNumber})
	stubRpc.SetResponse(fdgAddr, "rootClaim", rpcblock.Latest, nil, []interface{}{rootClaim})
	stubRpc.SetResponse(fdgAddr, "status", rpcblock.Latest, nil, []interface{}{uint8(status)})

	metadata, err := game.GetExtendedMetadata(context.Background(), rpcblock.Latest)
	require.NoError(t, err)
	require.Equal(t, contracts.GameMetadata{
		L1Head:        l1Head,
		L2SequenceNum: bigs.Uint64Strict(l2SequenceNumber),
		RootClaim:     rootClaim,
		Status:        status,
	}, metadata)

	claims, err := game.GetAllClaims(context.Background(), rpcblock.Latest)
	require.NoError(t, err)
	require.Empty(t, claims)
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
