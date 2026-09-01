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
	DoneValue     bool
	Dir           string
	PrestateErr   error
}

func (g *StubGamePlayer) ValidatePrestate(_ context.Context) error {
	return g.PrestateErr
}

func (g *StubGamePlayer) ProgressGame(_ context.Context, _ uint64) (types.GameStatus, bool) {
	g.ProgressCount++
	return g.StatusValue, g.DoneValue
}

func (g *StubGamePlayer) Status() types.GameStatus {
	return g.StatusValue
}

func (g *StubGamePlayer) Done() bool {
	return g.DoneValue
}
