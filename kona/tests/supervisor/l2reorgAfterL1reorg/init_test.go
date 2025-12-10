package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	spresets "github.com/op-rs/kona/supervisor/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// Other setups may be added here, hydrated from the same orchestrator
	presets.DoMain(m, spresets.WithSimpleInteropMinimal())
}
