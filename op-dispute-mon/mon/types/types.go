package types

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

type StatusBatch struct {
	InProgress    int
	DefenderWon   int
	ChallengerWon int
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

type ForecastBatch struct {
	AgreeDefenderAhead      int
	DisagreeDefenderAhead   int
	AgreeChallengerAhead    int
	DisagreeChallengerAhead int
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
