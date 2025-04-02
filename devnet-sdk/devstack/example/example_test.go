package example

import (
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/presets"
)

// TestExample1 starts an interop chain and verifies that the local unsafe head advances.
func TestExample1(t *testing.T) {
	sys := presets.NewSimpleInterop(t)

	sys.Supervisor.VerifySyncStatus(dsl.WithAllLocalUnsafeHeadsAdvancedBy(10))
}

// TODO(#15138): adjust sysgo / syskt to be graceful
//  when things already exist
//  (just set the shim with existing orchestrator-managed service)
//func TestExample2(t *testing.T) {
//	preset := presets.NewSimpleInterop(t)
//	preset.Log.Info("foobar 123")
//}
