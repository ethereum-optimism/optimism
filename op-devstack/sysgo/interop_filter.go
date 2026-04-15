package sysgo

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-interop-filter/filter"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// InteropFilter wraps an in-process op-interop-filter service for devstack.
// Follows the same pattern as OpSupervisor (supervisor_op.go).
type InteropFilter struct {
	mu      sync.Mutex
	name    string
	logger  log.Logger
	service *filter.Service
}

// HTTPEndpoint returns the service's actual RPC endpoint (e.g. "http://127.0.0.1:12345").
// The caller is responsible for proxying if needed.
func (f *InteropFilter) HTTPEndpoint() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.service.HTTPEndpoint()
}

// Ready returns true once all chain ingesters have backfilled.
func (f *InteropFilter) Ready() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.service == nil {
		return false
	}
	return f.service.Ready()
}

// WaitForReady blocks until all chain ingesters have backfilled or the timeout expires.
// Call this after the supernode/CL layer has started so that blocks are being produced.
func (f *InteropFilter) WaitForReady(t devtest.T, timeout time.Duration) {
	waitCtx, waitCancel := context.WithTimeout(t.Ctx(), timeout)
	defer waitCancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for !f.Ready() {
		select {
		case <-waitCtx.Done():
			t.Require().Fail("interop filter did not become ready within timeout")
		case <-ticker.C:
		}
	}
	f.logger.Info("Interop filter ready")
}

// SetFailsafeEnabled toggles failsafe mode directly on the backend.
// No admin RPC or JWT needed — this is the in-process advantage.
func (f *InteropFilter) SetFailsafeEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.service.SetFailsafeEnabled(enabled)
}

// FailsafeEnabled returns the current failsafe state.
func (f *InteropFilter) FailsafeEnabled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.service.FailsafeEnabled()
}

// Stop gracefully shuts down the interop filter service.
func (f *InteropFilter) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.service == nil {
		f.logger.Warn("InteropFilter already stopped")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	f.logger.Info("Closing interop filter")
	closeErr := f.service.Stop(ctx)
	f.logger.Info("Closed interop filter", "err", closeErr)
	f.service = nil
}

// startInteropFilter creates and starts an in-process interop filter.
// Rollup configs are passed as Go structs — no file serialization needed.
// The filter connects to EL nodes via the provided RPC URLs.
func startInteropFilter(
	t devtest.T,
	name string,
	l2RPCs []string,
	rollupConfigs map[eth.ChainID]*rollup.Config,
) *InteropFilter {
	logger := t.Logger().New("component", name)

	cfg := &filter.Config{
		L2RPCs:              l2RPCs,
		RollupConfigs:       rollupConfigs,
		DataDir:             t.TempDir(),
		BackfillDuration:    30 * time.Second,
		MessageExpiryWindow: 7 * 24 * 3600, // 7 days in seconds
		PollInterval:        500 * time.Millisecond,
		ValidationInterval:  200 * time.Millisecond,
		RPCAddr:             "127.0.0.1",
		RPCPort:             0, // Auto-assign
		Version:             "devstack",
	}

	t.Require().NoError(cfg.Check(), "invalid interop filter config")

	service, err := filter.NewService(context.Background(), cfg, logger)
	t.Require().NoError(err, "failed to create interop filter service")

	f := &InteropFilter{
		name:    name,
		logger:  logger,
		service: service,
	}

	logger.Info("Starting interop filter")
	err = service.Start(context.Background())
	t.Require().NoError(err, "failed to start interop filter")
	t.Cleanup(func() { f.Stop() })
	logger.Info("Started interop filter", "endpoint", service.HTTPEndpoint())

	return f
}
