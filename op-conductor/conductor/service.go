package conductor

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
)

// New creates a new OpConductor instance.
func New(ctx context.Context, cfg *Config, log log.Logger, version string) (*OpConductor, error) {
	panic("unimplemented")
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
type OpConductor struct{}

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
