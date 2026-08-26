package types

import (
	"bytes"
	"maps"
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
	types.FastGameType,
	types.AlphabetGameType,
	types.KailuaGameType,
}

var superRootGameTypes = []types.GameType{
	types.SuperPermissionedGameType,
	types.SuperAsteriscKonaGameType,
	types.SuperCannonKonaGameType,
	types.ZKDisputeGameType,
}

// EnrichedClaim extends the faultTypes.Claim with additional context.
type EnrichedClaim struct {
	faultTypes.Claim
	Resolved bool
}

// BondRecord describes the monitor's normalized accounting view of one deposited game bond.
type BondRecord struct {
	Depositor common.Address
	// Recipient is the payout recipient under the snapshot's bond distribution mode.
	Recipient common.Address
	Amount    *big.Int
	Resolved  bool
	// Forfeited reports whether normalized accounting counts Amount as lost by Depositor and won by Recipient.
	Forfeited bool
	// ChallengerBond reports whether this record represents a ZK game's challenger bond.
	ChallengerBond bool
}

// BondGameData contains normalized bond and DelayedWETH state.
type BondGameData struct {
	Bonds      []BondRecord
	Recipients map[common.Address]bool
	Credits    map[common.Address]*big.Int
	// ExpectedCredits maps recipients to the credit amounts expected under the snapshot's bond distribution mode.
	ExpectedCredits map[common.Address]*big.Int

	BondDistributionMode faultTypes.BondDistributionMode
	WithdrawalRequests   map[common.Address]*contracts.WithdrawalRequest
	WETHContract         common.Address
	WETHDelay            time.Duration
	ETHCollateral        *big.Int
	CreditWithdrawableAt time.Time
}

// RecipientAddresses returns a deterministic union of all addresses represented in the bond snapshot.
func (d *BondGameData) RecipientAddresses() []common.Address {
	recipients := make(map[common.Address]bool)
	for recipient := range d.Recipients {
		recipients[recipient] = true
	}
	for recipient := range d.Credits {
		recipients[recipient] = true
	}
	for recipient := range d.ExpectedCredits {
		recipients[recipient] = true
	}
	for recipient := range d.WithdrawalRequests {
		recipients[recipient] = true
	}
	for _, bond := range d.Bonds {
		recipients[bond.Depositor] = true
		if bond.Resolved {
			recipients[bond.Recipient] = true
		}
	}
	result := slices.Collect(maps.Keys(recipients))
	slices.SortFunc(result, func(a, b common.Address) int {
		return bytes.Compare(a[:], b[:])
	})
	return result
}

// EnrichedGame is a complete, pinned snapshot of a supported dispute game.
// Implementations are sealed so every game kind is handled explicitly.
type EnrichedGame interface {
	Common() *CommonGameData
	enrichedGame()
}

// BondedGame is a complete pinned snapshot of a game that holds bonds in DelayedWETH.
type BondedGame interface {
	EnrichedGame
	BondData() *BondGameData
	bondedGame()
}

// CommonGameData contains fields shared by every supported game kind.
type CommonGameData struct {
	types.GameMetadata
	LastUpdateTime   time.Time
	L1Head           common.Hash
	L1HeadNum        uint64
	L2SequenceNumber uint64
	RootClaim        common.Hash
	Status           types.GameStatus

	// AnchorStateRegistry is the address of the AnchorStateRegistry this game builds on.
	// Zero if the game's contract version does not expose it.
	AnchorStateRegistry common.Address

	AgreeWithClaim    bool
	ExpectedRootClaim common.Hash

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

// FaultGameData contains claims, bonds, withdrawals, and challenge state for a fault game.
type FaultGameData struct {
	CommonGameData
	BondGameData

	MaxClockDuration      uint64
	BlockNumberChallenged bool
	BlockNumberChallenger common.Address
	Claims                []EnrichedClaim
}

// ZKGameData contains proposal and parent state for a ZK dispute game.
type ZKGameData struct {
	CommonGameData
	BondGameData

	ParentIndex    uint32
	ParentStatus   *types.GameStatus
	ProposalStatus contracts.ProposalStatus
	Deadline       time.Time
	GameCreator    common.Address
	Challenger     common.Address
	Prover         common.Address
	TotalBonds     *big.Int
	ChallengerBond *big.Int
}

// SuperPermissionedGameData is the common snapshot of a SuperPermissioned game.
type SuperPermissionedGameData struct {
	CommonGameData
}

func (g *FaultGameData) Common() *CommonGameData             { return &g.CommonGameData }
func (g *ZKGameData) Common() *CommonGameData                { return &g.CommonGameData }
func (g *SuperPermissionedGameData) Common() *CommonGameData { return &g.CommonGameData }

func (*FaultGameData) enrichedGame()             {}
func (*ZKGameData) enrichedGame()                {}
func (*SuperPermissionedGameData) enrichedGame() {}

func (g *FaultGameData) BondData() *BondGameData { return &g.BondGameData }
func (g *ZKGameData) BondData() *BondGameData    { return &g.BondGameData }
func (*FaultGameData) bondedGame()               {}
func (*ZKGameData) bondedGame()                  {}

var (
	_ BondedGame = (*FaultGameData)(nil)
	_ BondedGame = (*ZKGameData)(nil)
)

// UsesOutputRoots returns true if the game type is one of the known types that use output roots as proposals.
func (g CommonGameData) UsesOutputRoots() bool {
	return slices.Contains(outputRootGameTypes, types.GameType(g.GameType))
}

// HasMixedAvailability returns true if some endpoints returned "not found" while others succeeded
// for this game. This indicates inconsistent block availability across the node network.
func (g CommonGameData) HasMixedAvailability() bool {
	if g.NodeEndpointTotalCount == 0 {
		return false
	}

	successfulEndpoints := g.NodeEndpointTotalCount - g.NodeEndpointErrorCount - g.NodeEndpointNotFoundCount - g.NodeEndpointOutOfSyncCount
	return g.NodeEndpointNotFoundCount > 0 && successfulEndpoints > 0
}

// HasMixedSafety returns true if some endpoints reported the root as safe and others as unsafe
// for this game. This indicates inconsistent safety assessment across the node network.
func (g CommonGameData) HasMixedSafety() bool {
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
