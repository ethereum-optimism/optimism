package types

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

type EnrichedGameData struct {
	types.GameMetadata

	L2BlockNumber uint64
	RootClaim     common.Hash
	Status        types.GameStatus
	ExpectedRoot  common.Hash
}

type StatusBatch struct {
	InProgress, DefenderWon, ChallengerWon int
}

func (s *StatusBatch) Add(status types.GameStatus) {
	switch status {
	case types.GameStatusInProgress:
		s.InProgress++
	case types.GameStatusDefenderWon:
		s.DefenderWon++
	case types.GameStatusChallengerWon:
		s.ChallengerWon++
	}
}

type DetectionBatch struct {
	InProgress             int
	AgreeDefenderWins      int
	DisagreeDefenderWins   int
	AgreeChallengerWins    int
	DisagreeChallengerWins int
}

func (d *DetectionBatch) Update(status types.GameStatus, agree bool) {
	switch status {
	case types.GameStatusInProgress:
		d.InProgress++
	case types.GameStatusDefenderWon:
		if agree {
			d.AgreeDefenderWins++
		} else {
			d.DisagreeDefenderWins++
		}
	case types.GameStatusChallengerWon:
		if agree {
			d.AgreeChallengerWins++
		} else {
			d.DisagreeChallengerWins++
		}
	}
}

func (d *DetectionBatch) Merge(other DetectionBatch) {
	d.InProgress += other.InProgress
	d.AgreeDefenderWins += other.AgreeDefenderWins
	d.DisagreeDefenderWins += other.DisagreeDefenderWins
	d.AgreeChallengerWins += other.AgreeChallengerWins
	d.DisagreeChallengerWins += other.DisagreeChallengerWins
}
