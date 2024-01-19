package types

import (
	"context"
	"fmt"
	"math/big"

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

type GameMetadata struct {
	GameType  uint8
	Timestamp uint64
	Proxy     common.Address
}

type LargePreimageIdent struct {
	Claimant common.Address
	UUID     *big.Int
}

type LargePreimageMetaData struct {
	LargePreimageIdent

	// Timestamp is the time at which the proposal first became fully available.
	// 0 when not all data is available yet
	Timestamp       uint64
	PartOffset      uint32
	ClaimedSize     uint32
	BlocksProcessed uint32
	BytesProcessed  uint32
	Countered       bool
}

type LargePreimageOracle interface {
	Addr() common.Address
	GetActivePreimages(ctx context.Context, blockHash common.Hash) ([]LargePreimageMetaData, error)
}
