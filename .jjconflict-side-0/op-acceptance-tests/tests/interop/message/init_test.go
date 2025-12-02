package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop())
}
