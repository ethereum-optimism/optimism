package fault

// Game is an interface that represents the state of a dispute game.
type Game interface {
	// Put adds a claim into the game state and returns its parent claim.
	Put(claim Claim) (Claim, error)

	// ClaimPairs returns a list of claim pairs.
	ClaimPairs() []struct {
		claim  Claim
		parent Claim
	}
}
