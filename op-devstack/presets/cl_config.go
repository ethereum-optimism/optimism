package presets

import (
	"github.com/HashKeyChain/verse/op-devstack/devtest"
	"github.com/HashKeyChain/verse/op-devstack/stack"
	"github.com/HashKeyChain/verse/op-devstack/sysgo"
	"github.com/HashKeyChain/verse/op-node/config"
	"github.com/HashKeyChain/verse/op-node/rollup/sync"
)

func WithExecutionLayerSyncOnVerifiers() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithL2CLOption(func(_ devtest.P, id stack.L2CLNodeID, cfg *config.Config) {
			// Can't enable ELSync on the sequencer or it will never start sequencing because
			// ELSync needs to receive gossip from the sequencer to drive the sync
			if !cfg.Driver.SequencerEnabled {
				cfg.Sync.SyncMode = sync.ELSync
			}
		}))
}

func WithConsensusLayerSync() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithL2CLOption(func(_ devtest.P, id stack.L2CLNodeID, cfg *config.Config) {
			cfg.Sync.SyncMode = sync.CLSync
		}))
}

func WithSafeDBEnabled() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithL2CLOption(func(p devtest.P, _ stack.L2CLNodeID, cfg *config.Config) {
			cfg.SafeDBPath = p.TempDir()
		}))
}
