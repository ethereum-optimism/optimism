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
	t.Setenv("DEVSTACK_L2CL_KIND", string(MixedL2CLKona))
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
					startStoppedSequencerCL(dt, nil, nil, network, nil, nil, nil, [32]byte{}, "node", opts)
				} else {
					startL2CLForKey(dt, nil, nil, network, nil, nil, nil, [32]byte{}, "node", "node", true, "", opts)
				}
			})
			require.Contains(t, dt.failure, "unsupported L2CL option for kona-node")
		})
	}
}
