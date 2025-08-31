package sysgo

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L2ELNode interface {
	hydrate(system stack.ExtensibleSystem)
	stack.Lifecycle
	UserRPC() string
	EngineRPC() string
	JWTPath() string
}

type L2ELConfig struct {
	SupervisorID *stack.SupervisorID
}

func L2ELWithSupervisor(supervisorID stack.SupervisorID) L2ELOption {
	return L2ELOptionFn(func(p devtest.P, id stack.L2ELNodeID, cfg *L2ELConfig) {
		cfg.SupervisorID = &supervisorID
	})
}

func DefaultL2ELConfig() *L2ELConfig {
	return &L2ELConfig{
		SupervisorID: nil,
	}
}

type L2ELOption interface {
	Apply(p devtest.P, id stack.L2ELNodeID, cfg *L2ELConfig)
}

// WithGlobalL2ELOption applies the L2ELOption to all L2ELNode instances in this orchestrator
func WithGlobalL2ELOption(opt L2ELOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.l2ELOptions = append(o.l2ELOptions, opt)
	})
}

type L2ELOptionFn func(p devtest.P, id stack.L2ELNodeID, cfg *L2ELConfig)

var _ L2ELOption = L2ELOptionFn(nil)

func (fn L2ELOptionFn) Apply(p devtest.P, id stack.L2ELNodeID, cfg *L2ELConfig) {
	fn(p, id, cfg)
}

// L2ELOptionBundle a list of multiple L2ELOption, to all be applied in order.
type L2ELOptionBundle []L2ELOption

var _ L2ELOption = L2ELOptionBundle(nil)

func (l L2ELOptionBundle) Apply(p devtest.P, id stack.L2ELNodeID, cfg *L2ELConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2ELOption")
		opt.Apply(p, id, cfg)
	}
}

// WithL2ELNode adds the default type of L2 CL node.
// The default can be configured with DEVSTACK_L2EL_KIND.
// Tests that depend on specific types can use options like WithKonaNode and WithOpNode directly.
func WithL2ELNode(id stack.L2ELNodeID, opts ...L2ELOption) stack.Option[*Orchestrator] {
	switch os.Getenv("DEVSTACK_L2EL_KIND") {
	case "op-reth":
		return WithOpReth(id, opts...)
	default:
		return WithOpGeth(id, opts...)
	}
}
