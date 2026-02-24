package core

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/op-rs/op-geth/proofs/utils"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// Other setups may be added here, hydrated from the same orchestrator
	presets.DoMain(m, utils.WithMixedOpProofPreset())
}
