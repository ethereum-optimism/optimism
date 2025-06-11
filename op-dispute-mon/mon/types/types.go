package types

import (
	"math/big"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

// outputRootGameTypes lists the set of legacy game types that use output roots
// It is assumed that all other game types use super roots
var outputRootGameTypes = []uint32{0, 1, 2, 3, 6, 254, 255, 1337}

// EnrichedClaim extends the faultTypes.Claim with additional context.
type EnrichedClaim struct {
	faultTypes.Claim
	Resolved bool
}

type EnrichedGameData struct {
	types.GameMetadata
	LastUpdateTime        time.Time
	L1Head                common.Hash
	L1HeadNum             uint64
	L2BlockNumber         uint64
	RootClaim             common.Hash
	Status                types.GameStatus
	MaxClockDuration      uint64
	BlockNumberChallenged bool
	BlockNumberChallenger common.Address
	Claims                []EnrichedClaim

	AgreeWithClaim    bool
	ExpectedRootClaim common.Hash

	// Recipients maps addresses to true if they are a bond recipient in the game.
	Recipients map[common.Address]bool

	// Credits records the paid out bonds for the game, keyed by recipient.
	Credits map[common.Address]*big.Int

	BondDistributionMode faultTypes.BondDistributionMode

	// WithdrawalRequests maps recipients with withdrawal requests in DelayedWETH for this game.
	WithdrawalRequests map[common.Address]*contracts.WithdrawalRequest

	// WETHContract is the address of the DelayedWETH contract used by this game
	// The contract is potentially shared by multiple games.
	WETHContract common.Address

	// WETHDelay is the delay applied before credits can be withdrawn.
	WETHDelay time.Duration

	// ETHCollateral is the ETH balance of the (potentially shared) WETHContract
	// This ETH balance will be used to pay out any bonds required by the games
	// that use the same DelayedWETH contract.
	ETHCollateral *big.Int
}

// UsesOutputRoots returns true if the game type is one of the known types that use output roots as proposals.
func (g EnrichedGameData) UsesOutputRoots() bool {
	return slices.Contains(outputRootGameTypes, g.GameType)
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
