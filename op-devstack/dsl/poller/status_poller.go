package poller

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// maxSyncStatusPollsInFlight bounds the number of concurrent sync-status calls,
// so a slow or hung RPC during a virtual-node swap cannot grow the goroutine set
// without limit while still keeping pressure on the route.
const maxSyncStatusPollsInFlight = 4

// StatusPoller repeatedly calls optimism_syncStatus on an L2 CL node in the
// background and classifies each response. It exists so an acceptance test can
// prove that patient callers never observe method-not-found or route-missing
// responses while the supernode swaps a chain's virtual node during reorg
// recovery, without the test having to manage RPC clients or goroutines itself.
type StatusPoller struct {
	t   devtest.T
	log log.Logger
	rpc client.RPC

	cancel context.CancelFunc
	done   <-chan struct{}

	success        atomic.Int64
	methodNotFound atomic.Int64
	notFound       atomic.Int64
	unavailable    atomic.Int64
}

// StartStatusPoller begins polling the CL node's sync status in the background.
// It registers a t.Cleanup() to stop the poller and wait for it to drain at the
// end of the test, so callers never manage its lifecycle directly.
func StartStatusPoller(cl *dsl.L2CLNode) *StatusPoller {
	t := cl.Escape().T()
	ctx, cancel := context.WithCancel(t.Ctx())
	done := make(chan struct{})

	p := &StatusPoller{
		t:      t,
		log:    cl.Escape().Logger(),
		rpc:    cl.Escape().ClientRPC(),
		cancel: cancel,
		done:   done,
	}
	t.Cleanup(p.stop)

	go p.run(ctx, done)
	return p
}

func (p *StatusPoller) run(ctx context.Context, done chan struct{}) {
	defer close(done)

	var wg sync.WaitGroup
	defer wg.Wait()
	inFlight := make(chan struct{}, maxSyncStatusPollsInFlight)

	poll := func() {
		select {
		case inFlight <- struct{}{}:
		case <-ctx.Done():
			return
		default:
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-inFlight }()

			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			var out eth.SyncStatus
			err := p.rpc.CallContext(callCtx, &out, "optimism_syncStatus")
			if err != nil && ctx.Err() != nil {
				return
			}
			p.record(err)
			if err != nil {
				p.log.Warn("supernode sync status poll failed", "err", err)
			}
		}()
	}

	poll()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

func (p *StatusPoller) record(err error) {
	if err == nil {
		p.success.Add(1)
		return
	}

	var rpcErr gethrpc.Error
	if errors.As(err, &rpcErr) && rpcErr.ErrorCode() == int(eth.MethodNotFound) {
		p.methodNotFound.Add(1)
		return
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "404") || strings.Contains(msg, "not found"):
		p.notFound.Add(1)
	case strings.Contains(msg, "503") || strings.Contains(msg, "service unavailable") || errors.Is(err, context.DeadlineExceeded):
		p.unavailable.Add(1)
	}
}

func (p *StatusPoller) total() int64 {
	return p.success.Load() +
		p.methodNotFound.Load() +
		p.notFound.Load() +
		p.unavailable.Load()
}

// WaitForNextSuccess blocks until the poller records a successful sync-status
// response after this call, failing the test if none arrives in time.
func (p *StatusPoller) WaitForNextSuccess() {
	baseline := p.success.Load()
	require.Eventually(p.t, func() bool {
		return p.success.Load() > baseline
	}, 10*time.Second, 200*time.Millisecond, "sync status poller should record a successful response")
}

// RequireNoRouteErrors asserts the poller observed at least one response and
// never saw a JSON-RPC method-not-found or a 404 route-missing response — i.e.
// the per-chain RPC gate held across the virtual-node swap.
func (p *StatusPoller) RequireNoRouteErrors() {
	require.Greater(p.t, p.total(), int64(0), "sync status poller should observe RPC calls")
	require.Zero(p.t, p.methodNotFound.Load(), "supernode route returned JSON-RPC method-not-found during reorg")
	require.Zero(p.t, p.notFound.Load(), "supernode route returned 404 during reorg")
}

func (p *StatusPoller) stop() {
	p.cancel()
	<-p.done
}
