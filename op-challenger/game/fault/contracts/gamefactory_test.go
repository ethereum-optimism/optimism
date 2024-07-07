package contracts

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var (
	factoryAddr = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
	batchSize   = 5
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
			stubRpc.SetResponse(factoryAddr, test.method, rpcblock.ByHash(blockHash), nil, []interface{}{test.result})
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
		Index:     0,
		GameType:  0,
		Timestamp: 1234,
		Proxy:     common.Address{0xaa},
	}
	game1 := types.GameMetadata{
		Index:     1,
		GameType:  1,
		Timestamp: 5678,
		Proxy:     common.Address{0xbb},
	}
	game2 := types.GameMetadata{
		Index:     2,
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

func TestGetAllGames(t *testing.T) {
	blockHash := common.Hash{0xbb, 0xce}
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	game0 := types.GameMetadata{
		Index:     0,
		GameType:  0,
		Timestamp: 1234,
		Proxy:     common.Address{0xaa},
	}
	game1 := types.GameMetadata{
		Index:     1,
		GameType:  1,
		Timestamp: 5678,
		Proxy:     common.Address{0xbb},
	}
	game2 := types.GameMetadata{
		Index:     2,
		GameType:  99,
		Timestamp: 9988,
		Proxy:     common.Address{0xcc},
	}

	expectedGames := []types.GameMetadata{game0, game1, game2}
	stubRpc.SetResponse(factoryAddr, methodGameCount, rpcblock.ByHash(blockHash), nil, []interface{}{big.NewInt(int64(len(expectedGames)))})
	for idx, expected := range expectedGames {
		expectGetGame(stubRpc, idx, blockHash, expected)
	}
	actualGames, err := factory.GetAllGames(context.Background(), blockHash)
	require.NoError(t, err)
	require.Equal(t, expectedGames, actualGames)
}

func TestGetAllGamesAtOrAfter(t *testing.T) {
	tests := []struct {
		gameCount       int
		earliestGameIdx int
	}{
		{gameCount: batchSize * 4, earliestGameIdx: batchSize + 3},
		{gameCount: 0, earliestGameIdx: 0},
		{gameCount: batchSize * 2, earliestGameIdx: batchSize},
		{gameCount: batchSize * 2, earliestGameIdx: batchSize + 1},
		{gameCount: batchSize * 2, earliestGameIdx: batchSize - 1},
		{gameCount: batchSize * 2, earliestGameIdx: batchSize * 2},
		{gameCount: batchSize * 2, earliestGameIdx: batchSize*2 + 1},
		{gameCount: batchSize - 2, earliestGameIdx: batchSize - 3},
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Count_%v_Start_%v", test.gameCount, test.earliestGameIdx), func(t *testing.T) {
			blockHash := common.Hash{0xbb, 0xce}
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			var allGames []types.GameMetadata
			for i := 0; i < test.gameCount; i++ {
				allGames = append(allGames, types.GameMetadata{
					Index:     uint64(i),
					GameType:  uint32(i),
					Timestamp: uint64(i),
					Proxy:     common.Address{byte(i)},
				})
			}

			stubRpc.SetResponse(factoryAddr, methodGameCount, rpcblock.ByHash(blockHash), nil, []interface{}{big.NewInt(int64(len(allGames)))})
			for idx, expected := range allGames {
				expectGetGame(stubRpc, idx, blockHash, expected)
			}
			// Set an earliest timestamp that's in the middle of a batch
			earliestTimestamp := uint64(test.earliestGameIdx)
			actualGames, err := factory.GetGamesAtOrAfter(context.Background(), blockHash, earliestTimestamp)
			require.NoError(t, err)
			// Games come back in descending timestamp order
			var expectedGames []types.GameMetadata
			if test.earliestGameIdx < len(allGames) {
				expectedGames = slices.Clone(allGames[test.earliestGameIdx:])
			}
			slices.Reverse(expectedGames)
			require.Equal(t, len(expectedGames), len(actualGames))
			if len(expectedGames) != 0 {
				// Don't assert equal for empty arrays, we accept nil or empty array
				require.Equal(t, expectedGames, actualGames)
			}
		})
	}
}

func TestGetGameFromParameters(t *testing.T) {
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	traceType := uint32(123)
	outputRoot := common.Hash{0x01}
	l2BlockNum := common.BigToHash(big.NewInt(456)).Bytes()
	stubRpc.SetResponse(
		factoryAddr,
		methodGames,
		rpcblock.Latest,
		[]interface{}{traceType, outputRoot, l2BlockNum},
		[]interface{}{common.Address{0xaa}, uint64(1)},
	)
	addr, err := factory.GetGameFromParameters(context.Background(), traceType, outputRoot, uint64(456))
	require.NoError(t, err)
	require.Equal(t, common.Address{0xaa}, addr)
}

func TestGetGameImpl(t *testing.T) {
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	gameType := faultTypes.CannonGameType
	gameImplAddr := common.Address{0xaa}
	stubRpc.SetResponse(
		factoryAddr,
		methodGameImpls,
		rpcblock.Latest,
		[]interface{}{gameType},
		[]interface{}{gameImplAddr})
	actual, err := factory.GetGameImpl(context.Background(), faultTypes.CannonGameType)
	require.NoError(t, err)
	require.Equal(t, gameImplAddr, actual)
}

func TestDecodeDisputeGameCreatedLog(t *testing.T) {
	_, factory := setupDisputeGameFactoryTest(t)
	fdgAbi := snapshots.LoadDisputeGameFactoryABI()
	eventAbi := fdgAbi.Events[eventDisputeGameCreated]
	gameAddr := common.Address{0x11}
	gameType := uint32(4)
	rootClaim := common.Hash{0xaa, 0xbb, 0xcc}

	createValidReceipt := func() *ethTypes.Receipt {
		return &ethTypes.Receipt{
			Status:          ethTypes.ReceiptStatusSuccessful,
			ContractAddress: fdgAddr,
			Logs: []*ethTypes.Log{
				{
					Address: fdgAddr,
					Topics: []common.Hash{
						eventAbi.ID,
						common.BytesToHash(gameAddr.Bytes()),
						common.BytesToHash(big.NewInt(int64(gameType)).Bytes()),
						rootClaim,
					},
				},
			},
		}
	}

	t.Run("IgnoreIncorrectContract", func(t *testing.T) {
		rcpt := createValidReceipt()
		rcpt.Logs[0].Address = common.Address{0xff}
		_, _, _, err := factory.DecodeDisputeGameCreatedLog(rcpt)
		require.ErrorIs(t, err, ErrEventNotFound)
	})

	t.Run("IgnoreInvalidEvent", func(t *testing.T) {
		rcpt := createValidReceipt()
		rcpt.Logs[0].Topics = rcpt.Logs[0].Topics[0:2]
		_, _, _, err := factory.DecodeDisputeGameCreatedLog(rcpt)
		require.ErrorIs(t, err, ErrEventNotFound)
	})

	t.Run("IgnoreWrongEvent", func(t *testing.T) {
		rcpt := createValidReceipt()
		rcpt.Logs[0].Topics = []common.Hash{
			fdgAbi.Events["ImplementationSet"].ID,
			common.BytesToHash(common.Address{0x11}.Bytes()), // Implementation addr
			common.BytesToHash(big.NewInt(4).Bytes()),        // Game type

		}
		// Check the log is a valid ImplementationSet
		name, _, err := factory.contract.DecodeEvent(rcpt.Logs[0])
		require.NoError(t, err)
		require.Equal(t, "ImplementationSet", name)

		_, _, _, err = factory.DecodeDisputeGameCreatedLog(rcpt)
		require.ErrorIs(t, err, ErrEventNotFound)
	})

	t.Run("ValidEvent", func(t *testing.T) {
		rcpt := createValidReceipt()
		actualGameAddr, actualGameType, actualRootClaim, err := factory.DecodeDisputeGameCreatedLog(rcpt)
		require.NoError(t, err)
		require.Equal(t, gameAddr, actualGameAddr)
		require.Equal(t, gameType, actualGameType)
		require.Equal(t, rootClaim, actualRootClaim)
	})
}

func expectGetGame(stubRpc *batchingTest.AbiBasedRpc, idx int, blockHash common.Hash, game types.GameMetadata) {
	stubRpc.SetResponse(
		factoryAddr,
		methodGameAtIndex,
		rpcblock.ByHash(blockHash),
		[]interface{}{big.NewInt(int64(idx))},
		[]interface{}{
			game.GameType,
			game.Timestamp,
			game.Proxy,
		})
}

func TestCreateTx(t *testing.T) {
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	traceType := uint32(123)
	outputRoot := common.Hash{0x01}
	l2BlockNum := common.BigToHash(big.NewInt(456)).Bytes()
	bond := big.NewInt(49284294829)
	stubRpc.SetResponse(factoryAddr, methodInitBonds, rpcblock.Latest, []interface{}{traceType}, []interface{}{bond})
	stubRpc.SetResponse(factoryAddr, methodCreateGame, rpcblock.Latest, []interface{}{traceType, outputRoot, l2BlockNum}, nil)
	tx, err := factory.CreateTx(context.Background(), traceType, outputRoot, uint64(456))
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
	require.NotNil(t, tx.Value)
	require.Truef(t, bond.Cmp(tx.Value) == 0, "Expected bond %v but was %v", bond, tx.Value)
}

func setupDisputeGameFactoryTest(t *testing.T) (*batchingTest.AbiBasedRpc, *DisputeGameFactoryContract) {
	fdgAbi := snapshots.LoadDisputeGameFactoryABI()

	stubRpc := batchingTest.NewAbiBasedRpc(t, factoryAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batchSize)
	factory := NewDisputeGameFactoryContract(metrics.NoopContractMetrics, factoryAddr, caller)
	return stubRpc, factory
}
