package fault

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
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

	// ClaimPairs returns a list of claim pairs.
	ClaimPairs() []struct {
		claim  Claim
		parent Claim
	}
}

// Node is a node in the game state tree.
type Node struct {
	self     Claim
	children []Node
}

// NodeKey is the key for a given list of [Claim] children.
type NodeKey struct {
	// Position is embedded to pass through its methods.
	Position
	// Value is the claim hash.
	Value common.Hash
}

// FromClaim creates a [NodeKey] from a given [Claim].
func FromClaim(claim *Claim) NodeKey {
	return NodeKey{
		Position: claim.Position,
		Value:    claim.Value,
	}
}

// gameState is a struct that represents the state of a dispute game.
// The game state implements the [Game] interface.
type gameState struct {
	nodes map[NodeKey]Node
}

// NewGameState returns a new game state.
func NewGameState() *gameState {
	return &gameState{
		nodes: make(map[NodeKey]Node),
	}
}

// getParent returns the parent of a given [Claim].
func (g *gameState) getParent(claim Claim) (Claim, error) {
	// Create a new NodeKey from the claim.
	key := FromClaim(&claim)

	// Get the node from the game state.
	node, ok := g.nodes[key]
	if !ok {
		return Claim{}, ErrClaimNotFound
	}

	// Return the parent.
	if node.self.Parent == nil {
		return Claim{}, ErrClaimNotFound
	}
	return *node.self.Parent, nil
}

// Put adds a claim into the game state.
func (g *gameState) Put(claim Claim) error {
	// Create a new NodeKey from the claim.
	key := FromClaim(&claim)

	// Check if the claim already exists.
	if _, ok := g.nodes[key]; ok {
		return ErrClaimExists
	}

	// Create a new node.
	node := Node{
		self:     claim,
		children: make([]Node, 0),
	}

	// Add the node to the game state.
	g.nodes[key] = node

	// Update any parent nodes.
	if claim.Parent != nil {
		g.addChild(*claim.Parent, node)
	}

	return nil
}

// addChild adds a node to parent [Claim].
func (g *gameState) addChild(parent Claim, child Node) {
	// Create a new NodeKey from the child's parent.
	parentKey := FromClaim(&parent)

	// Get the parent node.
	parentNode, ok := g.nodes[parentKey]
	if !ok {
		return
	}

	// Add the child to the parent node.
	parentNode.children = append(parentNode.children, child)
}

// ClaimPairs returns a list of claim pairs.
func (g *gameState) ClaimPairs() []struct {
	claim  Claim
	parent Claim
} {
	// Create a list of claim pairs.
	pairs := make([]struct {
		claim  Claim
		parent Claim
	}, 0)

	// Iterate over the game state.
	for _, node := range g.nodes {
		// Iterate over the node's children.
		for _, child := range node.children {
			// Append the claim pair.
			pairs = append(pairs, struct {
				claim  Claim
				parent Claim
			}{
				claim:  child.self,
				parent: node.self,
			})
		}
	}

	return pairs
}
