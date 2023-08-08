package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type CannonGameHelper struct {
	FaultGameHelper
}

func (g *CannonGameHelper) StartChallenger(ctx context.Context, l1Endpoint string, l2Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			c.GameAddress = g.addr
			c.GameDepth = cannonGameDepth
			c.TraceType = config.TraceTypeCannon
			c.AgreeWithProposedOutput = false
			c.CannonL2 = l2Endpoint
			c.CannonBin = "../cannon/bin/cannon"
			c.CannonDatadir = g.t.TempDir()
			c.CannonServer = "../op-program/bin/op-program"
			c.CannonAbsolutePreState = "../op-program/bin/prestate.json"
			c.CannonSnapshotFreq = config.DefaultCannonSnapshotFreq
		},
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
