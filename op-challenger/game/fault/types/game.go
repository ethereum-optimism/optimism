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

	// GetParent returns the parent of the provided claim.
	GetParent(claim Claim) (Claim, error)

	// IsDuplicate returns true if the provided [Claim] already exists in the game state
	// referencing the same parent claim
	IsDuplicate(claim Claim) bool

	// AgreeWithClaimLevel returns if the game state agrees with the provided claim level.
	AgreeWithClaimLevel(claim Claim) bool

	MaxDepth() uint64
}

type claimEntry struct {
	ClaimData
	ParentContractIndex int
}

type extendedClaim struct {
	self     Claim
	children []claimEntry
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	agreeWithProposedOutput bool
	root                    claimEntry
	// contractIndicies maps a contract index to it's extended claim.
	// This is used to perform O(1) parent lookups.
	contractIndicies map[int]*extendedClaim
	// claims maps a claim entry to it's extended claim.
	claims map[claimEntry]*extendedClaim
	depth  uint64
}

// NewGameState returns a new game state.
// The provided [Claim] is used as the root node.
func NewGameState(agreeWithProposedOutput bool, root Claim, depth uint64) *gameState {
	claims := make(map[claimEntry]*extendedClaim)
	parents := make(map[int]*extendedClaim)
	rootClaimEntry := makeClaimEntry(root)
	claims[rootClaimEntry] = &extendedClaim{
		self:     root,
		children: make([]claimEntry, 0),
	}
	parents[root.ContractIndex] = claims[rootClaimEntry]
	return &gameState{
		agreeWithProposedOutput: agreeWithProposedOutput,
		root:                    rootClaimEntry,
		claims:                  claims,
		contractIndicies:        parents,
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
	parent.children = append(parent.children, makeClaimEntry(claim))
	claimWithExtension := &extendedClaim{
		self:     claim,
		children: make([]claimEntry, 0),
	}
	g.claims[makeClaimEntry(claim)] = claimWithExtension
	g.contractIndicies[claim.ContractIndex] = claimWithExtension
	return nil
}

func (g *gameState) IsDuplicate(claim Claim) bool {
	_, ok := g.claims[makeClaimEntry(claim)]
	return ok
}

func (g *gameState) Claims() []Claim {
	queue := []claimEntry{g.root}
	var out []Claim
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		queue = append(queue, g.getChildren(item)...)
		out = append(out, g.claims[item].self)
	}
	return out
}

func (g *gameState) MaxDepth() uint64 {
	return g.depth
}

func (g *gameState) getChildren(c claimEntry) []claimEntry {
	return g.claims[c].children
}

func (g *gameState) GetParent(claim Claim) (Claim, error) {
	parent := g.getParent(claim)
	if parent == nil {
		return Claim{}, ErrClaimNotFound
	}
	return parent.self, nil
}

func (g *gameState) getParent(claim Claim) *extendedClaim {
	if claim.IsRoot() {
		return nil
	}
	if parent, ok := g.contractIndicies[claim.ParentContractIndex]; ok {
		return parent
	}
	return nil
}

func makeClaimEntry(claim Claim) claimEntry {
	return claimEntry{
		ClaimData:           claim.ClaimData,
		ParentContractIndex: claim.ParentContractIndex,
	}
}
