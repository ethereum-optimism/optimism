package withdrawals

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGameStateReader_GetGameStates(t *testing.T) {
	block := rpcblock.ByNumber(42)
	stubRpc := batchingTest.NewAbiBasedRpc(t, gameAddr, gameStateABI)
	stubRpc.AddContract(otherGameAddr, gameStateABI)
	stubRpc.SetResponse(gameAddr, methodStatus, block, nil, []interface{}{uint8(types.GameStatusChallengerWon)})
	stubRpc.SetResponse(gameAddr, methodL1Head, block, nil, []interface{}{gameL1Head})
	stubRpc.SetResponse(otherGameAddr, methodStatus, block, nil, []interface{}{uint8(types.GameStatusDefenderWon)})
	stubRpc.SetResponse(otherGameAddr, methodL1Head, block, nil, []interface{}{common.Hash{0x22}})

	reader := NewGameStateReader(batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize))
	states, err := reader.GetGameStates(context.Background(), block, []common.Address{gameAddr, otherGameAddr})

	require.NoError(t, err)
	require.Equal(t, []GameState{
		{Status: types.GameStatusChallengerWon, L1Head: gameL1Head},
		{Status: types.GameStatusDefenderWon, L1Head: common.Hash{0x22}},
	}, states)
}
