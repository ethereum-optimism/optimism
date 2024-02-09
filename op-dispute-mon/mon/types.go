package mon

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

type statusBatch struct {
	inProgress, defenderWon, challengerWon int
}

func (s *statusBatch) Add(status types.GameStatus) {
	switch status {
	case types.GameStatusInProgress:
		s.inProgress++
	case types.GameStatusDefenderWon:
		s.defenderWon++
	case types.GameStatusChallengerWon:
		s.challengerWon++
	}
}

type detectionBatch struct {
	inProgress             int
	agreeDefenderWins      int
	disagreeDefenderWins   int
	agreeChallengerWins    int
	disagreeChallengerWins int
}

func (d *detectionBatch) Update(status types.GameStatus, agree bool) {
	switch status {
	case types.GameStatusInProgress:
		d.inProgress++
	case types.GameStatusDefenderWon:
		if agree {
			d.agreeDefenderWins++
		} else {
			d.disagreeDefenderWins++
		}
	case types.GameStatusChallengerWon:
		if agree {
			d.agreeChallengerWins++
		} else {
			d.disagreeChallengerWins++
		}
	}
}

func (d *detectionBatch) Merge(other detectionBatch) {
	d.inProgress += other.inProgress
	d.agreeDefenderWins += other.agreeDefenderWins
	d.disagreeDefenderWins += other.disagreeDefenderWins
	d.agreeChallengerWins += other.agreeChallengerWins
	d.disagreeChallengerWins += other.disagreeChallengerWins
}
