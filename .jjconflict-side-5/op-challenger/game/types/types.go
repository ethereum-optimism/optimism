package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNotInSync       = errors.New("local node too far behind")
	ErrInvalidPrestate = errors.New("absolute prestate does not match")
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

type GameMetadata struct {
	Index     uint64
	GameType  uint32
	Timestamp uint64
	Proxy     common.Address
}

type SyncValidator interface {
	// ValidateNodeSynced checks that the local node is sufficiently up to date to play the game.
	// It returns client.ErrNotInSync if the node is too far behind.
	ValidateNodeSynced(ctx context.Context, gameL1Head eth.BlockID) error
}
