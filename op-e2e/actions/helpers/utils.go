package helpers

import (
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
)

// L1BlockTime is the L1 block time, and the beacon slot length, that the action helpers assume.
const L1BlockTime = 12

func DefaultRollupTestParams() *e2eutils.TestParams {
	return &e2eutils.TestParams{
		MaxSequencerDrift:   40,
		SequencerWindowSize: 120,
		ChannelTimeout:      120,
		L1BlockTime:         L1BlockTime,
		AllocType:           config.DefaultAllocType,
	}
}

var DefaultAlloc = &e2eutils.AllocParams{PrefundTestUsers: true}

type VerifierCfg struct {
	SafeHeadListener safeDB
}

type VerifierOpt func(opts *VerifierCfg)

func WithSafeHeadListener(l safeDB) VerifierOpt {
	return func(opts *VerifierCfg) {
		opts.SafeHeadListener = l
	}
}

func DefaultVerifierCfg() *VerifierCfg {
	return &VerifierCfg{
		SafeHeadListener: safedb.Disabled,
	}
}

type SequencerCfg struct {
	VerifierCfg
}

func DefaultSequencerConfig() *SequencerCfg {
	return &SequencerCfg{VerifierCfg: *DefaultVerifierCfg()}
}

type SequencerOpt func(opts *SequencerCfg)

func WithVerifierOpts(opts ...VerifierOpt) SequencerOpt {
	return func(cfg *SequencerCfg) {
		for _, opt := range opts {
			opt(&cfg.VerifierCfg)
		}
	}
}
