package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/disputemon"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/common"
)

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
	honestActors  []common.Address
}

type DisputeMonOption func(*disputeMonOptions)

func WithDisputeMonRollupNodes(nodes ...*dsl.L2CLNode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.rollupRPCs = append(opts.rollupRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func WithDisputeMonSupernodes(nodes ...dsl.SuperRootSource) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.UserRPC())
			}
		}
	}
}

func WithDisputeMonHonestActors(actors ...common.Address) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		opts.honestActors = append(opts.honestActors, actors...)
	}
}

func (s *SingleChainInterop) StartDisputeMon() *disputemon.DisputeMon {
	return StartDisputeMon(
		s.T,
		s.L1EL,
		s.L2ChainA.DisputeGameFactoryProxyAddr(),
		WithDisputeMonSupernodes(s.SuperRoots),
	)
}

func StartDisputeMon(
	t devtest.T,
	l1EL *dsl.L1ELNode,
	factory common.Address,
	options ...DisputeMonOption,
) *disputemon.DisputeMon {
	t.Require().NotNil(l1EL, "L1 EL is required to start dispute monitor")
	opts := &disputeMonOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}
	t.Require().NotEmpty(
		append(append([]string{}, opts.rollupRPCs...), opts.supernodeRPCs...),
		"at least one rollup node or supernode is required to start dispute monitor",
	)
	runtime := sysgo.StartDisputeMon(t, sysgo.DisputeMonConfig{
		L1RPC:              l1EL.Escape().UserRPC(),
		GameFactoryAddress: factory,
		RollupRPCs:         opts.rollupRPCs,
		SupernodeRPCs:      opts.supernodeRPCs,
		HonestActors:       opts.honestActors,
	})
	return disputemon.New(t, runtime.MetricsURL())
}
