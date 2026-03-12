package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2CLNode interface {
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

	// EnableReqRespSync is the flag to enable/disable req-resp sync.
	EnableReqRespSync bool

	// UseReqRespSync controls whether to use the req-resp sync protocol. EnableReqRespSync == false && UseReqRespSync == true is not allowed, and node will fail to start.
	UseReqRespSync bool

	// NoDiscovery is the flag to enable/disable discovery
	NoDiscovery bool

	FollowSource string
}

func L2CLSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IsSequencer = true
	})
}

func L2CLIndexing() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IndexingMode = true
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
		IndexingMode:      false,
		EnableReqRespSync: true,
		UseReqRespSync:    true,
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
