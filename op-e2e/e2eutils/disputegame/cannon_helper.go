package disputegame

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core"
)

type CannonGameHelper struct {
	FaultGameHelper
}

func (g *CannonGameHelper) StartChallenger(ctx context.Context, rollupCfg *rollup.Config, l2Genesis *core.Genesis, l1Endpoint string, l2Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			c.GameAddress = g.addr
			c.TraceType = config.TraceTypeCannon
			c.AgreeWithProposedOutput = false
			c.CannonL2 = l2Endpoint
			c.CannonBin = "../cannon/bin/cannon"
			c.CannonDatadir = g.t.TempDir()
			c.CannonServer = "../op-program/bin/op-program"
			c.CannonAbsolutePreState = "../op-program/bin/prestate.json"
			c.CannonSnapshotFreq = 10_000_000

			genesisBytes, err := json.Marshal(l2Genesis)
			g.require.NoError(err, "marshall l2 genesis config")
			genesisFile := filepath.Join(c.CannonDatadir, "l2-genesis.json")
			g.require.NoError(os.WriteFile(genesisFile, genesisBytes, 0644))
			c.CannonL2GenesisPath = genesisFile

			rollupBytes, err := json.Marshal(rollupCfg)
			g.require.NoError(err, "marshall rollup config")
			rollupFile := filepath.Join(c.CannonDatadir, "rollup.json")
			g.require.NoError(os.WriteFile(rollupFile, rollupBytes, 0644))
			c.CannonRollupConfigPath = rollupFile
		},
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
