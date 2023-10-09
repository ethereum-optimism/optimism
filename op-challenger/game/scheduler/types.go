package scheduler

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

type DiskManager interface {
	DirForGame(addr common.Address) string
	RemoveAllExcept(addrs []common.Address) error
}

type job struct {
	addr   common.Address
	player types.GamePlayer
	status types.GameStatus
}
