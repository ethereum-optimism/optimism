package withdrawals

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodStatus = "status"
	methodL1Head = "l1Head"
)

// gameStateABI covers the getters every dispute game implementation exposes, so game states can be
// read without detecting the version of each game.
var gameStateABI = mustParseABI(`[
	{"inputs":[],"name":"status","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"l1Head","outputs":[{"type":"bytes32"}],"stateMutability":"view","type":"function"}
]`)

type GameState struct {
	Status types.GameStatus
	// L1Head is the L1 block the game was created against, and so a lower bound on the block in
	// which it was created.
	L1Head common.Hash
}

type GameStateReader interface {
	GetGameStates(ctx context.Context, block rpcblock.Block, games []common.Address) ([]GameState, error)
}

type gameStateReader struct {
	caller *batching.MultiCaller
}

func NewGameStateReader(caller *batching.MultiCaller) GameStateReader {
	return &gameStateReader{caller: caller}
}

func (r *gameStateReader) GetGameStates(ctx context.Context, block rpcblock.Block, games []common.Address) ([]GameState, error) {
	calls := make([]batching.Call, 0, len(games)*2)
	for _, game := range games {
		calls = append(calls,
			batching.NewContractCall(gameStateABI, game, methodStatus),
			batching.NewContractCall(gameStateABI, game, methodL1Head))
	}
	results, err := r.caller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to load game states: %w", err)
	}
	states := make([]GameState, len(games))
	for i, game := range games {
		status, err := types.GameStatusFromUint8(results[i*2].GetUint8(0))
		if err != nil {
			return nil, fmt.Errorf("invalid status for game %v: %w", game, err)
		}
		states[i] = GameState{Status: status, L1Head: results[i*2+1].GetHash(0)}
	}
	return states, nil
}

func mustParseABI(json string) *abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader([]byte(json)))
	if err != nil {
		panic(err)
	}
	return &parsed
}
