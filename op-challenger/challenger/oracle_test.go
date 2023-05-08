package challenger

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
)

// TestCreateDisputeGame_Fails tests that the createDisputeGame function
// is not implemented.
func TestCreateDisputeGame_Fails(t *testing.T) {
	challenger := &Challenger{}

	// Create a valid dispute game
	_, err := challenger.createDisputeGame(
		context.Background(), // Context
		flags.GameType(0),    // Attestation Dispute Game
		&eth.Bytes32{},       // Output Root
		big.NewInt(0),        // L2 Block Number
	)

	if err.Error() != "dispute game creation not implemented" {
		t.Errorf("expected error: dispute game creation not implemented, got: %v", err)
	}
}
