package scheduler

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type GamePlayer interface {
	ValidatePrestate(ctx context.Context) error
	// ProgressGame acts on the game as required at the given L1 head, returning the game's status and
	// whether it requires no further work beyond that status.
	ProgressGame(ctx context.Context, l1Head eth.BlockID) (types.GameStatus, bool)
	Status() types.GameStatus
	Done() bool
}

type DiskManager interface {
	DirForGame(addr common.Address) string
	RemoveAllExcept(addrs []common.Address) error
}

type job struct {
	block  eth.BlockID
	addr   common.Address
	player GamePlayer
	status types.GameStatus
	done   bool
}

func newJob(block eth.BlockID, addr common.Address, player GamePlayer, status types.GameStatus) *job {
	return &job{
		block:  block,
		addr:   addr,
		player: player,
		status: status,
	}
}
