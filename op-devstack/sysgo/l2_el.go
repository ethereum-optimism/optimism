package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L2ELNode interface {
	stack.Lifecycle
	UserRPC() string
	EngineRPC() string
	JWTPath() string
}

type L2ELConfig struct {
	P2PAddr       string
	P2PPort       int
	P2PNodeKeyHex string
	StaticPeers   []string
	TrustedPeers  []string
	ProofHistory  bool
}

func L2ELWithProofHistory(enable bool) L2ELOption {
	return L2ELOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2ELConfig) {
		cfg.ProofHistory = enable
	})
}

// L2ELWithP2PConfig sets deterministic P2P identity and static peers for the L2 EL.
func L2ELWithP2PConfig(addr string, port int, nodeKeyHex string, staticPeers, trustedPeers []string) L2ELOption {
	return L2ELOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2ELConfig) {
		cfg.P2PAddr = addr
		cfg.P2PPort = port
		cfg.P2PNodeKeyHex = nodeKeyHex
		cfg.StaticPeers = staticPeers
		cfg.TrustedPeers = trustedPeers
	})
}

func DefaultL2ELConfig() *L2ELConfig {
	return &L2ELConfig{
		P2PAddr:       "127.0.0.1",
		P2PPort:       0,
		P2PNodeKeyHex: "",
		StaticPeers:   nil,
		TrustedPeers:  nil,
		ProofHistory:  false,
	}
}

type L2ELOption interface {
	Apply(p devtest.T, target ComponentTarget, cfg *L2ELConfig)
}

type L2ELOptionFn func(p devtest.T, target ComponentTarget, cfg *L2ELConfig)

var _ L2ELOption = L2ELOptionFn(nil)

func (fn L2ELOptionFn) Apply(p devtest.T, target ComponentTarget, cfg *L2ELConfig) {
	fn(p, target, cfg)
}

// L2ELOptionBundle a list of multiple L2ELOption, to all be applied in order.
type L2ELOptionBundle []L2ELOption

var _ L2ELOption = L2ELOptionBundle(nil)

func (l L2ELOptionBundle) Apply(p devtest.T, target ComponentTarget, cfg *L2ELConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2ELOption")
		opt.Apply(p, target, cfg)
	}
}
