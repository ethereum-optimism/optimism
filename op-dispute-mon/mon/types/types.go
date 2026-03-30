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
var outputRootGameTypes = []types.GameType{
	types.CannonGameType,
	types.PermissionedGameType,
	types.AsteriscGameType,
	types.AsteriscKonaGameType,
	types.OPSuccinctGameType,
	types.CannonKonaGameType,
	types.ZKDisputeGameType,
	types.FastGameType,
	types.AlphabetGameType,
	types.KailuaGameType,
}

var superRootGameTypes = []types.GameType{
	types.SuperCannonGameType,
	types.SuperPermissionedGameType,
	types.SuperAsteriscKonaGameType,
	types.SuperCannonKonaGameType,
}

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
	L2SequenceNumber      uint64
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

	// NodeEndpointErrors stores endpoint IDs that returned errors other than "not found" for this game.
	NodeEndpointErrors map[string]bool

	// NodeEndpointErrorCount tracks the total number of errors for this game across all endpoints.
	NodeEndpointErrorCount int

	// NodeEndpointNotFoundCount tracks the number of endpoints that returned "not found" for this game.
	NodeEndpointNotFoundCount int

	// NodeEndpointOutOfSyncCount tracks the number of endpoints that were out of sync for this game.
	NodeEndpointOutOfSyncCount int

	// NodeEndpointTotalCount tracks the total number of endpoints attempted for this game.
	NodeEndpointTotalCount int

	// NodeEndpointSafeCount tracks the number of endpoints that reported the root as safe.
	NodeEndpointSafeCount int

	// NodeEndpointUnsafeCount tracks the number of endpoints that reported the root as unsafe.
	NodeEndpointUnsafeCount int

	// NodeEndpointDifferentRoots tracks whether endpoints returned different roots for this game.
	// For output root games, this means different output roots. For super root games, different super roots.
	NodeEndpointDifferentRoots bool
}

// UsesOutputRoots returns true if the game type is one of the known types that use output roots as proposals.
func (g EnrichedGameData) UsesOutputRoots() bool {
	return slices.Contains(outputRootGameTypes, types.GameType(g.GameType))
}

// HasMixedAvailability returns true if some endpoints returned "not found" while others succeeded
// for this game. This indicates inconsistent block availability across the node network.
func (g EnrichedGameData) HasMixedAvailability() bool {
	if g.NodeEndpointTotalCount == 0 {
		return false
	}

	successfulEndpoints := g.NodeEndpointTotalCount - g.NodeEndpointErrorCount - g.NodeEndpointNotFoundCount
	return g.NodeEndpointNotFoundCount > 0 && successfulEndpoints > 0
}

// HasMixedSafety returns true if some endpoints reported the root as safe and others as unsafe
// for this game. This indicates inconsistent safety assessment across the node network.
func (g EnrichedGameData) HasMixedSafety() bool {
	return g.NodeEndpointSafeCount > 0 && g.NodeEndpointUnsafeCount > 0
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
