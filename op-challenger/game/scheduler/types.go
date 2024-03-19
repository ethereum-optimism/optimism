package scheduler

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

type GamePlayer interface {
	ValidatePrestate(ctx context.Context) error
	ProgressGame(ctx context.Context) types.GameStatus
	Status() types.GameStatus
}

type DiskManager interface {
	DirForGame(addr common.Address) string
	RemoveAllExcept(addrs []common.Address) error
}

type job struct {
	block  uint64
	addr   common.Address
	player GamePlayer
	status types.GameStatus
}

func newJob(block uint64, addr common.Address, player GamePlayer, status types.GameStatus) *job {
	return &job{
		block:  block,
		addr:   addr,
		player: player,
		status: status,
	}
}
