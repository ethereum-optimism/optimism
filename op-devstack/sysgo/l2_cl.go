package sysgo

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2CLNode interface {
	hydrate(system stack.ExtensibleSystem)
	stack.Lifecycle
	UserRPC() string
	InteropRPC() (endpoint string, jwtSecret eth.Bytes32)
}

type L2CLConfig struct {
	// SyncMode to run, if this is a sequencer
	SequencerSyncMode nodeSync.Mode
	// SyncMode to run, if this is a verifier
	VerifierSyncMode nodeSync.Mode

	// SafeDBPath is the path to the safe DB to use. Disabled if empty.
	SafeDBPath string

	IsSequencer  bool
	IndexingMode bool
}

func L2CLSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
		cfg.IsSequencer = true
	})
}

func L2CLIndexing() L2CLOption {
	return L2CLOptionFn(func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
		cfg.IndexingMode = true
	})
}

func DefaultL2CLConfig() *L2CLConfig {
	return &L2CLConfig{
		SequencerSyncMode: nodeSync.CLSync,
		VerifierSyncMode:  nodeSync.CLSync,
		SafeDBPath:        "",
		IsSequencer:       false,
		IndexingMode:      false,
	}
}

type L2CLOption interface {
	Apply(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig)
}

// WithGlobalL2CLOption applies the L2CLOption to all L2CLNode instances in this orchestrator
func WithGlobalL2CLOption(opt L2CLOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.l2CLOptions = append(o.l2CLOptions, opt)
	})
}

type L2CLOptionFn func(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig)

var _ L2CLOption = L2CLOptionFn(nil)

func (fn L2CLOptionFn) Apply(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
	fn(p, id, cfg)
}

// L2CLOptionBundle a list of multiple L2CLOption, to all be applied in order.
type L2CLOptionBundle []L2CLOption

var _ L2CLOption = L2CLOptionBundle(nil)

func (l L2CLOptionBundle) Apply(p devtest.P, id stack.L2CLNodeID, cfg *L2CLConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2CLOption")
		opt.Apply(p, id, cfg)
	}
}

// WithL2CLNode adds the default type of L2 CL node.
// The default can be configured with DEVSTACK_L2CL_KIND.
// Tests that depend on specific types can use options like WithKonaNode and WithOpNode directly.
func WithL2CLNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	switch os.Getenv("DEVSTACK_L2CL_KIND") {
	case "kona":
		return WithKonaNode(l2CLID, l1CLID, l1ELID, l2ELID, opts...)
	default:
		return WithOpNode(l2CLID, l1CLID, l1ELID, l2ELID, opts...)
	}
}
