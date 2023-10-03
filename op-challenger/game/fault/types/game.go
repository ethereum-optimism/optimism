package types

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// ErrClaimExists is returned when a claim already exists in the game state.
	ErrClaimExists = errors.New("claim exists in game state")

	// ErrClaimNotFound is returned when a claim does not exist in the game state.
	ErrClaimNotFound = errors.New("claim not found in game state")
)

// Game is an interface that represents the state of a dispute game.
type Game interface {
	// Put adds a claim into the game state.
	Put(claim Claim) error

	// PutAll adds a list of claims into the game state.
	PutAll(claims []Claim) error

	// Claims returns all of the claims in the game.
	Claims() []Claim

	// GetParent returns the parent of the provided claim.
	GetParent(claim Claim) (Claim, error)

	// IsDuplicate returns true if the provided [Claim] already exists in the game state
	// referencing the same parent claim
	IsDuplicate(claim Claim) bool

	// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
	AgreeWithClaimLevel(claim Claim) bool

	MaxDepth() uint64
}

type claimID common.Hash

func computeClaimID(claim Claim) claimID {
	return claimID(crypto.Keccak256Hash(
		new(big.Int).SetUint64(claim.Position.ToGIndex()).Bytes(),
		claim.Value.Bytes(),
		big.NewInt(int64(claim.ParentContractIndex)).Bytes(),
	))
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	agreeWithProposedOutput bool
	// claims is the list of claims in the same order as the contract
	claims   []Claim
	claimIDs map[claimID]bool
	depth    uint64
}

// NewGameState returns a new game state.
// The provided [Claim] is used as the root node.
func NewGameState(agreeWithProposedOutput bool, root Claim, depth uint64) *gameState {
	claimIDs := make(map[claimID]bool)
	claimIDs[computeClaimID(root)] = true
	return &gameState{
		agreeWithProposedOutput: agreeWithProposedOutput,
		claims:                  []Claim{root},
		claimIDs:                claimIDs,
		depth:                   depth,
	}
}

// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
func (g *gameState) AgreeWithClaimLevel(claim Claim) bool {
	isOddLevel := claim.Depth()%2 == 1
	// If we agree with the proposed output, we agree with odd levels
	// If we disagree with the proposed output, we agree with the root claim level & even levels
	if g.agreeWithProposedOutput {
		return isOddLevel
	} else {
		return !isOddLevel
	}
}

// PutAll adds a list of claims into the [Game] state.
// If any of the claims already exist in the game state, an error is returned.
func (g *gameState) PutAll(claims []Claim) error {
	for _, claim := range claims {
		if err := g.Put(claim); err != nil {
			return err
		}
	}
	return nil
}

// Put adds a claim into the game state.
func (g *gameState) Put(claim Claim) error {
	if claim.IsRoot() || g.IsDuplicate(claim) {
		return ErrClaimExists
	}

	parent := g.getParent(claim)
	if parent == nil {
		return errors.New("no parent claim")
	}

	g.claims = append(g.claims, claim)
	g.claimIDs[computeClaimID(claim)] = true
	return nil
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
