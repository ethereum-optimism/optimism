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

func TestDisputeGameFactorySimpleGetters(t *testing.T) {
	blockNum := uint64(23)
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
				return game.GetGameCount(context.Background(), blockNum)
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.method, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			stubRpc.SetResponse(test.method, batching.BlockByNumber(blockNum), nil, []interface{}{test.result})
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
	blockNum := uint64(23)
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
		expectGetGame(stubRpc, idx, blockNum, expected)
		actual, err := factory.GetGame(context.Background(), uint64(idx), blockNum)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func expectGetGame(stubRpc *batchingTest.AbiBasedRpc, idx int, blockNum uint64, game types.GameMetadata) {
	stubRpc.SetResponse(
		methodGameAtIndex,
		batching.BlockByNumber(blockNum),
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
	address := common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAbi, address)
	caller := batching.NewMultiCaller(stubRpc, 100)
	factory, err := NewDisputeGameFactoryContract(address, caller)
	require.NoError(t, err)
	return stubRpc, factory
}
