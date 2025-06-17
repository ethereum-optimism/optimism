package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type ClaimData struct {
	ParentIndex *big.Int
	CounteredBy common.Address
	Claimant    common.Address
	Bond        *big.Int
	Claim       common.Hash
	Position    *big.Int
	Clock       *big.Int
}

type Claim struct {
	Value common.Hash
	Bond  *big.Int
	types.Position
	CounteredBy         common.Address
	Claimant            common.Address
	Clock               types.Clock
	ParentContractIndex int
}

func (d ClaimData) Decode() Claim {
	return Claim{
		Value:               d.Claim,
		Position:            types.NewPositionFromGIndex(d.Position),
		Bond:                d.Bond,
		CounteredBy:         d.CounteredBy,
		Claimant:            d.Claimant,
		Clock:               types.DecodeClock(d.Clock),
		ParentContractIndex: int(d.ParentIndex.Int64()),
	}
}

type FaultDisputeGame struct {
	// IDisputeGame.sol read methods
	L1Head func() TypedCall[common.Hash] `sol:"l1Head"`

	// IFaultDisputeGame.sol read methods
	ClaimDataLen    func() TypedCall[*big.Int]                                                 `sol:"claimDataLen"`
	ClaimData       func(*big.Int) TypedCall[ClaimData]                                        `sol:"claimData"`
	GetRequiredBond func(position *Uint128) TypedCall[*big.Int]                                `sol:"getRequiredBond"`
	MaxGameDepth    func() TypedCall[*big.Int]                                                 `sol:"maxGameDepth"`
	SplitDepth      func() TypedCall[*big.Int]                                                 `sol:"splitDepth"`
	SubGame         func(parentClaimIndex *big.Int, subGameIndex *big.Int) TypedCall[*big.Int] `sol:"subgame"`

	// IFaultDisputeGame.sol write methods
	Move   func(targetClaim common.Hash, targetClaimIndex *big.Int, newClaim common.Hash, isAttack bool) TypedCall[any] `sol:"move"`
	Attack func(targetClaim common.Hash, targetClaimIndex *big.Int, counterClaim common.Hash) TypedCall[any]            `sol:"attack"`
	Defend func(targetClaim common.Hash, targetClaimIndex *big.Int, supportingClaim common.Hash) TypedCall[any]         `sol:"defend"`
}

func NewFaultDisputeGame(opts ...CallFactoryOption) *FaultDisputeGame {
	fdg := NewBindings[FaultDisputeGame](opts...)
	return &fdg
}
