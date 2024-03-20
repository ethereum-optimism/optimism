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
	L1Head        common.Hash
	L1HeadNum     uint64
	L2BlockNumber uint64
	RootClaim     common.Hash
	Status        types.GameStatus
	Duration      uint64
	Claims        []faultTypes.Claim

	// Credits records the paid out bonds for the game, keyed by recipient.
	Credits map[common.Address]*big.Int

	// WETHContract is the address of the DelayedWETH contract used by this game
	// The contract is potentially shared by multiple games.
	WETHContract common.Address

	// ETHCollateral is the ETH balance of the (potentially shared) WETHContract
	// This ETH balance will be used to pay out any bonds required by the games that use the same DelayedWETH contract.
	ETHCollateral *big.Int
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

type ForecastBatch struct {
	AgreeDefenderAhead      int
	DisagreeDefenderAhead   int
	AgreeChallengerAhead    int
	DisagreeChallengerAhead int

	AgreeDefenderWins      int
	DisagreeDefenderWins   int
	AgreeChallengerWins    int
	DisagreeChallengerWins int
}
