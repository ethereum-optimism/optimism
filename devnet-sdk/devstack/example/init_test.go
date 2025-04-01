package example

import (
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/presets"
)

// TestMain ensures the orchestrator is setup correctly for this package.
func TestMain(m *testing.M) {
	presets.DoMain(m)
}
