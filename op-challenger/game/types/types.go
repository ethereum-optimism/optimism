package types

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type GameStatus uint8

const (
	GameStatusInProgress GameStatus = iota
	GameStatusChallengerWon
	GameStatusDefenderWon
)

// String returns the string representation of the game status.
func (s GameStatus) String() string {
	switch s {
	case GameStatusInProgress:
		return "In Progress"
	case GameStatusChallengerWon:
		return "Challenger Won"
	case GameStatusDefenderWon:
		return "Defender Won"
	default:
		return "Unknown"
	}
}

// GameStatusFromUint8 returns a game status from the uint8 representation.
func GameStatusFromUint8(i uint8) (GameStatus, error) {
	if i > 2 {
		return GameStatus(i), fmt.Errorf("invalid game status: %d", i)
	}
	return GameStatus(i), nil
}

type PlayerCreator interface {
	Addr() common.Address
	Create(dir string) (GamePlayer, error)
}

type GamePlayer interface {
	Addr() common.Address
	ProgressGame(ctx context.Context) GameStatus
	Status() GameStatus
}
