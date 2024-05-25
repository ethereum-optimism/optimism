package types

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// ErrClaimNotFound is returned when a claim does not exist in the game state.
	ErrClaimNotFound = errors.New("claim not found in game state")
)

// Game is an interface that represents the state of a dispute game.
type Game interface {
	// Claims returns all of the claims in the game.
	Claims() []Claim

	// GetParent returns the parent of the provided claim.
	GetParent(claim Claim) (Claim, error)

	// DefendsParent returns true if and only if the claim is a defense (i.e. goes right) of
	// its parent.
	DefendsParent(claim Claim) bool

	// IsDuplicate returns true if the provided [Claim] already exists in the game state
	// referencing the same parent claim
	IsDuplicate(claim Claim) bool

	// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
	AgreeWithClaimLevel(claim Claim, agreeWithRootClaim bool) bool

	MaxDepth() uint64
}

type claimID common.Hash

func computeClaimID(claim Claim) claimID {
	return claimID(crypto.Keccak256Hash(
		claim.Position.ToGIndex().Bytes(),
		claim.Value.Bytes(),
		big.NewInt(int64(claim.ParentContractIndex)).Bytes(),
	))
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	// claims is the list of claims in the same order as the contract
	claims   []Claim
	claimIDs map[claimID]bool
	depth    uint64
}

// NewGameState returns a new game state.
// The provided [Claim] is used as the root node.
func NewGameState(claims []Claim, depth uint64) *gameState {
	claimIDs := make(map[claimID]bool)
	for _, claim := range claims {
		claimIDs[computeClaimID(claim)] = true
	}
	return &gameState{
		claims:   claims,
		claimIDs: claimIDs,
		depth:    depth,
	}
}

// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
func (g *gameState) AgreeWithClaimLevel(claim Claim, agreeWithRootClaim bool) bool {
	isOddLevel := claim.Depth()%2 == 1
	// If we agree with the proposed output, we agree with odd levels
	// If we disagree with the proposed output, we agree with the root claim level & even levels
	if agreeWithRootClaim {
		return !isOddLevel
	} else {
		return isOddLevel
	}
}

func (g *gameState) IsDuplicate(claim Claim) bool {
	return g.claimIDs[computeClaimID(claim)]
}

func (g *gameState) Claims() []Claim {
	// Defensively copy to avoid modifications to the underlying array.
	return append([]Claim(nil), g.claims...)
}

func (g *gameState) MaxDepth() uint64 {
	return g.depth
}

func (g *gameState) GetParent(claim Claim) (Claim, error) {
	parent := g.getParent(claim)
	if parent == nil {
		return Claim{}, ErrClaimNotFound
	}
	return *parent, nil
}

func (g *gameState) DefendsParent(claim Claim) bool {
	parent := g.getParent(claim)
	if parent == nil {
		return false
	}
	return claim.RightOf(parent.Position)
}

func (g *gameState) getParent(claim Claim) *Claim {
	if claim.IsRoot() {
		return nil
	}
	if claim.ParentContractIndex >= len(g.claims) || claim.ParentContractIndex < 0 {
		return nil
	}
	parent := g.claims[claim.ParentContractIndex]
	return &parent
}
