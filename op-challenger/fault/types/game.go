package types

import (
	"errors"
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

	// IsDuplicate returns true if the provided [Claim] already exists in the game state.
	IsDuplicate(claim Claim) bool

	// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
	AgreeWithClaimLevel(claim Claim) bool
}

type extendedClaim struct {
	self     Claim
	children []ClaimData
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	agreeWithProposedOutput bool
	root                    ClaimData
	claims                  map[ClaimData]*extendedClaim
	depth                   uint64
}

// NewGameState returns a new game state.
// The provided [Claim] is used as the root node.
func NewGameState(agreeWithProposedOutput bool, root Claim, depth uint64) *gameState {
	claims := make(map[ClaimData]*extendedClaim)
	claims[root.ClaimData] = &extendedClaim{
		self:     root,
		children: make([]ClaimData, 0),
	}
	return &gameState{
		agreeWithProposedOutput: agreeWithProposedOutput,
		root:                    root.ClaimData,
		claims:                  claims,
		depth:                   depth,
	}
}

// AgreeWithLevel returns if the game state agrees with the provided claim level.
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
	parent, ok := g.claims[claim.Parent]
	if !ok {
		return errors.New("no parent claim")
	} else {
		parent.children = append(parent.children, claim.ClaimData)
	}
	g.claims[claim.ClaimData] = &extendedClaim{
		self:     claim,
		children: make([]ClaimData, 0),
	}
	return nil
}

func (g *gameState) IsDuplicate(claim Claim) bool {
	_, ok := g.claims[claim.ClaimData]
	return ok
}

func (g *gameState) Claims() []Claim {
	queue := []ClaimData{g.root}
	var out []Claim
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		queue = append(queue, g.getChildren(item)...)
		out = append(out, g.claims[item].self)
	}
	return out
}

func (g *gameState) getChildren(c ClaimData) []ClaimData {
	return g.claims[c].children
}

func (g *gameState) getParent(claim Claim) (Claim, error) {
	if claim.IsRoot() {
		return Claim{}, ErrClaimNotFound
	}
	if parent, ok := g.claims[claim.Parent]; !ok {
		return Claim{}, ErrClaimNotFound
	} else {
		return parent.self, nil
	}
}
