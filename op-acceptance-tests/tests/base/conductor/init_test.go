package conductor

import (
	"testing"

	"github.com/HashKeyChain/verse/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimalWithConductors())
}
