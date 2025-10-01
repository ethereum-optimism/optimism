package base_multi

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates a two-L2 setup against the shared backend
func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithTwoL2())
}
