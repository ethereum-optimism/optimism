package fault

import (
	"errors"
)

var (
	// ErrClaimExists is returned when a claim already exists in the game state.
	ErrClaimExists = errors.New("claim exists in game state")
)

// Game is an interface that represents the state of a dispute game.
type Game interface {
	// Put adds a claim into the game state and returns its parent claim.
	Put(claim Claim) (Claim, error)

	// ClaimPairs returns a list of claim pairs.
	ClaimPairs() []struct {
		claim  Claim
		parent Claim
	}

	// Retrieve the parent of a given claim.
	GetClaimParent(claim Claim) Claim
}

// gameState is a struct that implements the [Game] interface.
type gameState struct {
	// claims is a map of claim gindex to
	claims map[uint64]Claim
}

// NewGameState returns a new game state.
func NewGameState() *gameState {
	return &gameState{
		claims: make(map[uint64]Claim),
	}
}

func (g *gameState) GetClaimParent(claim Claim) Claim {
	// If the claim is the root, we don't need to return a parent.
	parent := Claim{}
	if claim.Depth() != 0 {
		parent = g.claims[claim.GetParentGIndex()]
	}
	return parent
}

// Put adds a claim into the game state and returns its parent claim.
func (g *gameState) Put(claim Claim) (Claim, error) {
	// Check if the claim already exists.
	if _, ok := g.claims[claim.ToGIndex()]; ok {
		return Claim{}, ErrClaimExists
	}

	// Get the parent claim.
	parent := g.GetClaimParent(claim)

	// Add the claim to the game state.
	g.claims[claim.ToGIndex()] = claim

	return parent, nil
}

func (g *gameState) ClaimPairs() []struct {
	claim  Claim
	parent Claim
} {
	pairs := make([]struct {
		claim  Claim
		parent Claim
	}, 0, len(g.claims))
	for _, claim := range g.claims {
		pairs = append(pairs, struct {
			claim  Claim
			parent Claim
		}{
			claim:  claim,
			parent: g.GetClaimParent(claim),
		})
	}
	return pairs
}
