package conductor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// New creates a new OpConductor instance.
func New(ctx context.Context, cfg *Config, log log.Logger, version string) (*OpConductor, error) {
	if err := cfg.Check(); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	oc := &OpConductor{
		log:     log,
		version: version,
		cfg:     cfg,
	}

	err := oc.init(ctx)
	if err != nil {
		log.Error("failed to initialize OpConductor", "err", err)
		// ensure we always close the resources if we fail to initialize the conductor.
		if closeErr := oc.Stop(ctx); closeErr != nil {
			return nil, multierror.Append(err, closeErr)
		}
	}

	return oc, nil
}

func (c *OpConductor) init(ctx context.Context) error {
	c.log.Info("initializing OpConductor", "version", c.version)
	if err := c.initSequencerControl(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize sequencer control")
	}
	if err := c.initConsensus(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize consensus")
	}
	return nil
}

func (c *OpConductor) initSequencerControl(ctx context.Context) error {
	ec, err := opclient.NewRPC(ctx, c.log, c.cfg.ExecutionRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create geth rpc client")
	}
	gethCfg := sources.L2ClientDefaultConfig(&c.cfg.RollupCfg, true)
	// TODO: Add metrics tracer here. tracked by https://github.com/ethereum-optimism/protocol-quest/issues/45
	geth, err := sources.NewEthClient(ec, c.log, nil, &gethCfg.EthClientConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create geth client")
	}

	nc, err := opclient.NewRPC(ctx, c.log, c.cfg.NodeRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create node rpc client")
	}
	node := sources.NewRollupClient(nc)
	c.ctrl = client.NewSequencerControl(geth, node)
	return nil
}

func (c *OpConductor) initConsensus(ctx context.Context) error {
	serverAddr := fmt.Sprintf("%s:%d", c.cfg.ConsensusAddr, c.cfg.ConsensusPort)
	cons, err := consensus.NewRaftConsensus(c.log, c.cfg.RaftServerID, serverAddr, c.cfg.RaftStorageDir, c.cfg.RaftBootstrap, &c.cfg.RollupCfg)
	if err != nil {
		return errors.Wrap(err, "failed to create raft consensus")
	}
	c.cons = cons
	return nil
}

// OpConductor represents a full conductor instance and its resources, it does:
//  1. performs health checks on sequencer
//  2. participate in consensus protocol for leader election
//  3. and control sequencer state based on leader and sequencer health status.
//
// OpConductor has three states:
//  1. running: it is running normally, which executes control loop and participates in leader election.
//  2. paused: control loop (sequencer start/stop) is paused, but it still participates in leader election.
//     it is paused for disaster recovery situation
//  3. stopped: it is stopped, which means it is not participating in leader election and control loop. OpConductor cannot be started again from stopped mode.
type OpConductor struct {
	log     log.Logger
	version string
	cfg     *Config

	ctrl client.SequencerControl
	cons consensus.Consensus
}

var _ cliapp.Lifecycle = (*OpConductor)(nil)

// Start implements cliapp.Lifecycle.
func (*OpConductor) Start(ctx context.Context) error {
	panic("unimplemented")
}

// Stop implements cliapp.Lifecycle.
func (*OpConductor) Stop(ctx context.Context) error {
	panic("unimplemented")
}

// Stopped implements cliapp.Lifecycle.
func (*OpConductor) Stopped() bool {
	panic("unimplemented")
}
