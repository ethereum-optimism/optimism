package types

import "github.com/ethereum/go-ethereum/common"

type HonestActors map[common.Address]bool // Map for efficient lookup

func NewHonestActors(honestActors []common.Address) HonestActors {
	actors := make(map[common.Address]bool)
	for _, actor := range honestActors {
		actors[actor] = true
	}
	return actors
}

func (h HonestActors) Contains(addr common.Address) bool {
	return h[addr]
}
