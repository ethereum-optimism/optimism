package types

import (
	"math/big"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

// ResolvedBondAmount is the uint128 value where a bond is considered claimed.
var ResolvedBondAmount = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

type EnrichedGameData struct {
	types.GameMetadata
	L2BlockNumber uint64
	RootClaim     common.Hash
	Status        types.GameStatus
	Duration      uint64
	Claims        []faultTypes.Claim
}

// BidirectionalTree is a tree of claims represented as a flat list of claims.
// This keeps the tree structure identical to how claims are stored in the contract.
type BidirectionalTree struct {
	Claims []*BidirectionalClaim
}

type BidirectionalClaim struct {
	Claim    *faultTypes.Claim
	Children []*BidirectionalClaim
}

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
