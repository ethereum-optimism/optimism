package virtual_node

import (
	"context"
	"errors"
	"sync"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

// defaultInnerNodeFactory is the default factory that creates a real op-node
func defaultInnerNodeFactory(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error) {
	var overrides rollupNode.InitializationOverrides
	if initOverload != nil {
		overrides = *initOverload
	}
	return rollupNode.NewWithOverride(ctx, cfg, log, appVersion, m, overrides)
}

var (
	ErrVirtualNodeConfigNil      = errors.New("virtual node config is nil")
	ErrVirtualNodeAlreadyRunning = errors.New("virtual node already running")
	ErrVirtualNodeNotRunning     = errors.New("virtual node not running")
	ErrVirtualNodeCantStart      = errors.New("virtual node cannot be started in this state")
)

type VirtualNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type innerNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type innerNodeFactory func(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error)

type VNState int

const (
	VNStateNotStarted VNState = iota
	VNStateRunning
	VNStateStopped
)

type simpleVirtualNode struct {
	log        gethlog.Logger
	vnID       string
	appVersion string

	inner            innerNode                           // Inner node instance
	cfg              *opnodecfg.Config                   // op-node config for the virtual node
	initOverload     *rollupNode.InitializationOverrides // Shared resources which are overridden by the supernode
	innerNodeFactory innerNodeFactory                    // Factory function to create inner node (overloadable for testing)

	mu     sync.Mutex         // Protects state transitions
	state  VNState            // Current lifecycle state
	cancel context.CancelFunc // Cancels the running context
}

func generateVirtualNodeID() string {
	return uuid.New().String()[:4]
}

func NewVirtualNode(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) *simpleVirtualNode {
	vnID := generateVirtualNodeID()
	l := log.New("chain_id", cfg.Rollup.L2ChainID.String(), "vn_id", vnID)
	return &simpleVirtualNode{
		vnID:             vnID,
		cfg:              cfg,
		log:              l,
		initOverload:     initOverload,
		appVersion:       appVersion,
		innerNodeFactory: defaultInnerNodeFactory,
		state:            VNStateNotStarted,
	}
}

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	// Accquire lock while setting up inner node
	v.mu.Lock()
	if v.state != VNStateNotStarted {
		v.mu.Unlock()
		v.log.Debug("virtual node not in a valid state to start", "state", v.state)
		return ErrVirtualNodeCantStart
	}
	if v.cfg == nil {
		v.mu.Unlock()
		return ErrVirtualNodeConfigNil
	}

	runCtx, cancel := context.WithCancel(ctx)
	v.cancel = cancel

	// Capture inner node errors via cancel callback
	var innerErr error
	v.cfg.Cancel = func(err error) {
		innerErr = err
		cancel() // Cancel the run context when inner node fails
	}

	// Create and start the inner node
	m := opmetrics.NewMetrics("supernode")
	n, err := v.innerNodeFactory(runCtx, v.cfg, v.log, v.appVersion, m, v.initOverload)
	if err != nil {
		v.state = VNStateStopped
		v.mu.Unlock()
		return err
	}
	v.inner = n
	// Release the lock once the inner node is created
	v.state = VNStateRunning
	v.mu.Unlock()
	// Don't hold the lock while running or waiting for inner node to stop

	// Run inner node in goroutine
	// and await any signal to exit (Stop(), parent ctx, or inner error)
	go func() {
		innerErr = v.inner.Start(runCtx)
	}()
	<-runCtx.Done()

	// Clean up with lock to end of function
	v.mu.Lock()
	defer v.mu.Unlock()
	v.state = VNStateStopped
	v.cancel = nil

	// Stop the inner node if it's still running
	if v.inner != nil {
		stopCtx := context.Background()
		if err := v.inner.Stop(stopCtx); err != nil {
			v.log.Error("error stopping inner node", "err", err)
		}
	}

	// Return inner error if that's what caused the cancellation, otherwise context error
	if innerErr != nil {
		v.log.Warn("virtual node stopped due to inner error", "err", innerErr)
		return innerErr
	}
	if ctx.Err() != nil {
		v.log.Warn("virtual node stopped due to context cancellation", "err", ctx.Err())
		return ctx.Err()
	}
	v.log.Info("virtual node stopped")
	return nil
}

func (v *simpleVirtualNode) Stop(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.state != VNStateRunning {
		return nil // Already stopped or not started
	}

	// Cancel the run context to trigger shutdown
	if v.cancel != nil {
		v.cancel()
	}

	return nil
}

// State returns the current state of the virtual node (for testing and monitoring)
func (v *simpleVirtualNode) State() VNState {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state
}
