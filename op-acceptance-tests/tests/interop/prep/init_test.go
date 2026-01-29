package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	// Configure a system without interop activation.
	// We just want to test if we can sync with a supervisor before interop is configured.
	// The supervisor will not be indexing data yet before interop, and has to handle that interop is not scheduled.
	presets.DoMain(m, presets.WithSimpleInterop(), presets.WithUnscheduledInterop())
}
