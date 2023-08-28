package scheduler

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type GamePlayer interface {
	ProgressGame(ctx context.Context) bool
}

type DiskManager interface {
	DirForGame(addr common.Address) string
	RemoveAllExcept(addrs []common.Address) error
}

type job struct {
	addr     common.Address
	player   GamePlayer
	resolved bool
}
