package sysgo

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

type L2CLNode interface {
	stack.Lifecycle
	UserRPC() string
}

type L2CLConfig struct {
	// SyncMode, if this is a sequencer. kona-node takes it as --syncmode, which starts a
	// sequencer the same way in either mode.
	SequencerSyncMode nodeSync.Mode
	// SyncMode, if this is a verifier. kona-node takes it as --syncmode: in CL sync a verifier
	// derives from L1 without waiting for an unsafe payload when its EL is already synced, but
	// unlike op-node it has no unsafe-payload queue, so a gossiped payload with a gap still
	// makes the EL sync.
	VerifierSyncMode nodeSync.Mode
	// VerifierL1Confs delays L1 input to derivation, independently of sequencing.
	VerifierL1Confs uint64

	// SafeDBPath is the path to the safe DB to use. Disabled if empty.
	SafeDBPath string

	IsSequencer bool

	// ConductorRPC configures this node to use the conductor at the given RPC
	// endpoint. Empty disables conductor integration.
	ConductorRPC        string
	ConductorRPCTimeout time.Duration

	// NoDiscovery is the flag to enable/disable discovery
	NoDiscovery bool

	FollowSource string

	// OffsetELSafe retracts safe and finalized from the EL-sync tip by floor(OffsetELSafe / L2BlockTime) blocks.
	OffsetELSafe time.Duration

	// AltDAServerURL is the DA server the node reads alt-DA inputs from. Empty disables alt-DA.
	AltDAServerURL string
}

func L2CLSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IsSequencer = true
	})
}

func L2CLFollowSource(source string) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.FollowSource = source
	})
}

func DefaultL2CLConfig() *L2CLConfig {
	return &L2CLConfig{
		SequencerSyncMode: nodeSync.CLSync,
		VerifierSyncMode:  nodeSync.CLSync,
		SafeDBPath:        "",
		IsSequencer:       false,
		NoDiscovery:       false,
		FollowSource:      "",
	}
}

type L2CLOption interface {
	Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig)
}

type L2CLOptionFn func(p devtest.T, target ComponentTarget, cfg *L2CLConfig)

var _ L2CLOption = L2CLOptionFn(nil)

func (fn L2CLOptionFn) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	fn(p, target, cfg)
}

// L2CLOptionBundle a list of multiple L2CLOption, to all be applied in order.
type L2CLOptionBundle []L2CLOption

var _ L2CLOption = L2CLOptionBundle(nil)

func (l L2CLOptionBundle) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2CLOption")
		opt.Apply(p, target, cfg)
	}
}
