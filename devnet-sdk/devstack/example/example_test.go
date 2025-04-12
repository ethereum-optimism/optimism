package example

import (
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
)

// TestExample1 starts an interop chain and verifies that the local unsafe head advances.
func TestExample1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := SimpleInterop(t)

	sys.Supervisor.VerifySyncStatus(dsl.WithAllLocalUnsafeHeadsAdvancedBy(10))
}

func TestExample2(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := SimpleInterop(t)

	sys.Supervisor.VerifySyncStatus(dsl.WithAllLocalUnsafeHeadsAdvancedBy(4))
}
