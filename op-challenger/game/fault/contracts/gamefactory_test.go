package contracts

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	factoryAddr = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
)

func TestDisputeGameFactorySimpleGetters(t *testing.T) {
	blockHash := common.Hash{0xbb, 0xcd}
	tests := []struct {
		method   string
		args     []interface{}
		result   interface{}
		expected interface{} // Defaults to expecting the same as result
		call     func(game *DisputeGameFactoryContract) (any, error)
	}{
		{
			method:   methodGameCount,
			result:   big.NewInt(9876),
			expected: uint64(9876),
			call: func(game *DisputeGameFactoryContract) (any, error) {
				return game.GetGameCount(context.Background(), blockHash)
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.method, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			stubRpc.SetResponse(factoryAddr, test.method, batching.BlockByHash(blockHash), nil, []interface{}{test.result})
			status, err := test.call(factory)
			require.NoError(t, err)
			expected := test.expected
			if expected == nil {
				expected = test.result
			}
			require.Equal(t, expected, status)
		})
	}
}

func TestLoadGame(t *testing.T) {
	blockHash := common.Hash{0xbb, 0xce}
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	game0 := types.GameMetadata{
		GameType:  0,
		Timestamp: 1234,
		Proxy:     common.Address{0xaa},
	}
	game1 := types.GameMetadata{
		GameType:  1,
		Timestamp: 5678,
		Proxy:     common.Address{0xbb},
	}
	game2 := types.GameMetadata{
		GameType:  99,
		Timestamp: 9988,
		Proxy:     common.Address{0xcc},
	}
	expectedGames := []types.GameMetadata{game0, game1, game2}
	for idx, expected := range expectedGames {
		expectGetGame(stubRpc, idx, blockHash, expected)
		actual, err := factory.GetGame(context.Background(), uint64(idx), blockHash)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func expectGetGame(stubRpc *batchingTest.AbiBasedRpc, idx int, blockHash common.Hash, game types.GameMetadata) {
	stubRpc.SetResponse(
		factoryAddr,
		methodGameAtIndex,
		batching.BlockByHash(blockHash),
		[]interface{}{big.NewInt(int64(idx))},
		[]interface{}{
			game.GameType,
			game.Timestamp,
			game.Proxy,
		})
}

func setupDisputeGameFactoryTest(t *testing.T) (*batchingTest.AbiBasedRpc, *DisputeGameFactoryContract) {
	fdgAbi, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, factoryAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, 100)
	factory, err := NewDisputeGameFactoryContract(factoryAddr, caller)
	require.NoError(t, err)
	return stubRpc, factory
}
