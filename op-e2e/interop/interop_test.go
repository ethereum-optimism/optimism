package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
)

// TestInterop stands up a basic L1
// and multiple L2 states
func TestInterop(t *testing.T) {
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}

	s2 := system2{recipe: &recipe}
	s2.prepare(t)

	ids := s2.getL2IDs()

	netA := ids[0]
	// netB := ids[1]

	// getting a batcher for network A
	_ = s2.getBatcher(netA)
	// or by direct map access
	_ = s2.l2s[netA].batcher

	// TODO (placeholder) Let the system test-run for a bit
	time.Sleep(time.Second * 30)
}
