package lokahi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSafeHeadHoldsAcrossLokahiRestart checks that restarting lokahi never moves a node's safe
// head back: lokahi answers the heads it verified before the restart as soon as it is up again,
// and the safe heads advance from there.
func TestSafeHeadHoldsAcrossLokahiRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}

	// Verified history first, so a lokahi verifying again from activation would answer lower
	// heads for a while after the restart.
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 10, 120))
	}
	dsl.CheckAll(t, checks...)

	watch := watchSafeHeads(t, nodes)
	before := verifiedHeads(t, sys.Lokahi, nodes)
	sys.Lokahi.Stop()
	sys.Lokahi.Start()
	after := verifiedHeads(t, sys.Lokahi, nodes)

	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 60))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "lokahi restarted", nodes)

	errs := []error{watch.stop()}
	for _, node := range nodes {
		b, a := before[node], after[node]
		t.Logger().Info("Verified head across the restart", "chain", node.ChainID(), "before", b, "after", a)
		if a.Source != "verified" || a.Timestamp < b.Timestamp || a.Block.Number < b.Block.Number {
			errs = append(errs, fmt.Errorf("chain %s: lokahi answered %+v right after the restart, below %+v before it",
				node.ChainID(), a, b))
		}
	}
	t.Require().NoError(errors.Join(errs...))
}

// verifierHead is lokahi's answer to interop_verifiedHead and interop_finalizedHead.
type verifierHead struct {
	Block     eth.BlockID `json:"block"`
	Timestamp uint64      `json:"timestamp"`
	Source    string      `json:"source"`
}

// verifiedHeads asks lokahi for each node's chain's verified head.
func verifiedHeads(t devtest.T, lokahi *sysgo.Lokahi, nodes []*dsl.L2CLNode) map[*dsl.L2CLNode]verifierHead {
	ctx, cancel := context.WithTimeout(t.Ctx(), 30*time.Second)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	heads := make(map[*dsl.L2CLNode]verifierHead, len(nodes))
	for _, node := range nodes {
		var head verifierHead
		chainID, ok := node.ChainID().Uint64()
		t.Require().True(ok, "chain ID fits in 64 bits")
		t.Require().NoError(client.CallContext(ctx, &head, "interop_verifiedHead", chainID), "interop_verifiedHead")
		heads[node] = head
	}
	return heads
}

// safeHeadWatch polls nodes' safe heads until stopped and records the first time one decreases.
type safeHeadWatch struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	err    error
}

func watchSafeHeads(t devtest.T, nodes []*dsl.L2CLNode) *safeHeadWatch {
	ctx, cancel := context.WithCancel(t.Ctx())
	w := &safeHeadWatch{cancel: cancel}
	for _, node := range nodes {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			var last uint64
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				status, err := node.Escape().RollupAPI().SyncStatus(ctx)
				if err != nil {
					continue
				}
				safe := status.SafeL2.Number
				if safe < last {
					t.Logger().Warn("Safe head moved back", "node", node.String(), "from", last, "to", safe)
					w.fail(fmt.Errorf("%s: safe head moved back from %d to %d", node, last, safe))
				}
				last = safe
			}
		}()
	}
	return w
}

func (w *safeHeadWatch) fail(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
	}
}

// stop ends the watch and returns the first decrease seen, if any.
func (w *safeHeadWatch) stop() error {
	w.cancel()
	w.wg.Wait()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
