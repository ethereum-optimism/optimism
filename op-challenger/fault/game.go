package fault

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
}

// Node is a node in the game state tree.
type Node struct {
	self     Claim
	children []*Node
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	root   Node
	claims map[ClaimData]Claim
}

// NewGameState returns a new game state.
// The provided [Claim] is used as the root node.
func NewGameState(root Claim) *gameState {
	claims := make(map[ClaimData]Claim)
	claims[root.ClaimData] = root
	return &gameState{
		root: Node{
			self:     root,
			children: make([]*Node, 0),
		},
		claims: claims,
	}
}

// getParent returns the parent of the provided [Claim].
func (g *gameState) getParent(claim Claim) (Claim, error) {
	// If the claim is the root node, return an error.
	if claim.IsRoot() {
		return Claim{}, ErrClaimNotFound
	}

	// Walk down the tree from the root node to find the parent.
	found, err := g.recurseTree(&g.root, claim.Parent)
	if err != nil {
		return Claim{}, err
	}

	// Return the parent of the found node.
	return found.self, nil
}

// recurseTree recursively walks down the tree from the root node to find the
// node with the provided [Claim].
func (g *gameState) recurseTree(treeNode *Node, claim ClaimData) (*Node, error) {
	// Check if the current node is the claim.
	if treeNode.self.ClaimData == claim {
		return treeNode, nil
	}

	// Check all children of the current node.
	for _, child := range treeNode.children {
		// Recurse and drop errors.
		n, _ := g.recurseTree(child, claim)
		if n != nil {
			return n, nil
		}
	}

	// If we reach this point, the claim was not found.
	return nil, ErrClaimNotFound
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
	// The game is always initialized with a root claim. Cannot add a second.
	if claim.IsRoot() {
		return ErrClaimExists
	}

	// Grab the claim's parent.
	parent := claim.Parent

	// Walk down the tree from the root node to find the parent.
	found, err := g.recurseTree(&g.root, parent)
	if err != nil {
		return err
	}

	// Check that the node is not already in the tree.
	for _, child := range found.children {
		if child.self == claim {
			return ErrClaimExists
		}
	}

	// Create a new node.
	node := Node{
		self:     claim,
		children: make([]*Node, 0),
	}

	// Add the node to the tree.
	found.children = append(found.children, &node)
	g.claims[claim.ClaimData] = claim

	return nil
}

func (g *gameState) IsDuplicate(claim Claim) bool {
	_, ok := g.claims[claim.ClaimData]
	return ok
}

func (g *gameState) Claims() []Claim {
	return g.root.claims()
}

func (n *Node) claims() []Claim {
	var out []Claim
	out = append(out, n.self)
	for _, c := range n.children {
		out = append(out, c.claims()...)
	}
	return out
}
