package presets

import (

	"strings"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	devtestmetrics "github.com/ethereum-optimism/optimism/op-devstack/devtest/metrics"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
)

const disputeMonMetricPollInterval = 100 * time.Millisecond

type DisputeMon struct {
	t       devtest.T
	metrics *devtestmetrics.MetricsClient
}

func newDisputeMon(t devtest.T, metricsURL string) *DisputeMon {
	httpClient := client.NewBasicHTTPClient(strings.TrimRight(metricsURL, "/"), t.Logger())
	return &DisputeMon{
		t:       t,
		metrics: devtestmetrics.NewMetricsClient(httpClient),
	}
}

func (d *DisputeMon) VerifyGameCount(gameType gameTypes.GameType, expected int) {
	err := d.metrics.WaitForGauge(d.t.Ctx(), devtestmetrics.GaugeDefinition{
		Name:     "op_dispute_mon_games",
		Labels:   map[string]string{"game_type": gameType.String()},
		Expected: float64(expected),
	}, disputeMonMetricPollInterval)
	d.t.Require().NoError(err, "expected dispute monitor to export %d games of type %s", expected, gameType)
}

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
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

func WithDisputeMonSupernodes(nodes ...*dsl.Supernode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func (s *SingleChainInterop) StartDisputeMon() *DisputeMon {
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
) *DisputeMon {
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
	})
	return newDisputeMon(t, runtime.MetricsURL())
}

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
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

func WithDisputeMonSupernodes(nodes ...*dsl.Supernode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func StartDisputeMon(
	t devtest.T,
	l1EL *dsl.L1ELNode,
	factory common.Address,
	options ...DisputeMonOption,
) *DisputeMon {
	t.Helper()
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
	})
	return newDisputeMon(t, runtime.MetricsURL())
}
