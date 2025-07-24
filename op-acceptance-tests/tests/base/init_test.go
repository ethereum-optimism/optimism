package base

import (
	"testing"

	"github.com/HashKeyChain/verse/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal())
}
