package test

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

type StubGamePlayer struct {
	Addr          common.Address
	ProgressCount int
	StatusValue   types.GameStatus
	Dir           string
	PrestateErr   error
	ValidateCount int
}

func (g *StubGamePlayer) ValidatePrestate(_ context.Context) error {
	g.ValidateCount++
	return g.PrestateErr
}

func (g *StubGamePlayer) ProgressGame(_ context.Context) types.GameStatus {
	g.ProgressCount++
	return g.StatusValue
}

func (g *StubGamePlayer) Status() types.GameStatus {
	return g.StatusValue
}
