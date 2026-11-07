package sysgo

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestKonaOptionsRejectedBeforeStartup(t *testing.T) {
	// The env asks for op-node, so a kona-node rejection shows the explicit kind picks the CL.
	t.Setenv("DEVSTACK_L2CL_KIND", string(MixedL2CLOpNode))
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	for _, stopped := range []bool{false, true} {
		t.Run(fmt.Sprintf("stopped=%t", stopped), func(t *testing.T) {
			dt := &rejectedKonaOptionTest{T: devtest.SerialT(t)}
			network := &L2Network{chainID: eth.ChainIDFromUInt64(901)}
			opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
				cfg.OffsetELSafe = time.Second
			})}
			// Nil startup dependencies ensure validation runs before starting any process or RPC.
			require.PanicsWithValue(t, "Kona option rejected", func() {
				if stopped {
					startStoppedSequencerCL(dt, nil, nil, network, nil, nil, nil, [32]byte{}, MixedL2CLKona, "node", opts)
				} else {
					startL2CLForKey(dt, nil, nil, network, nil, nil, nil, [32]byte{}, MixedL2CLKona, "node", "node", true, "", opts)
				}
			})
			require.Contains(t, dt.failure, "unsupported L2CL option for kona-node")
		})
	}
}

// A kind the starters do not know, including the empty string, fails before anything starts
// instead of silently running op-node.
func TestUnknownL2CLKindRejectedBeforeStartup(t *testing.T) {
	for _, kind := range []MixedL2CLKind{"", "kona", "Kona-Node"} {
		for _, stopped := range []bool{false, true} {
			t.Run(fmt.Sprintf("kind=%q/stopped=%t", kind, stopped), func(t *testing.T) {
				dt := &rejectedKonaOptionTest{T: devtest.SerialT(t)}
				network := &L2Network{chainID: eth.ChainIDFromUInt64(901)}
				require.PanicsWithValue(t, "Kona option rejected", func() {
					if stopped {
						startStoppedSequencerCL(dt, nil, nil, network, nil, nil, nil, [32]byte{}, kind, "node", nil)
					} else {
						startL2CLForKey(dt, nil, nil, network, nil, nil, nil, [32]byte{}, kind, "node", "node", true, "", nil)
					}
				})
				require.Contains(t, dt.failure, "unknown L2 CL kind")
			})
		}
	}
}
