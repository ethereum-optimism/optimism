package lokahi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSafeHeadHoldsAcrossLokahiRestart checks that restarting lokahi never moves a node's safe
// head back: once lokahi caught up after the restart it answers at least the heads it verified
// before, and the safe heads advance from there.
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

// verifierHead is a head in lokahi's answer to interop_chainStatus.
type verifierHead struct {
	Block     eth.BlockID `json:"block"`
	Timestamp uint64      `json:"timestamp"`
	Source    string      `json:"source"`
	L1        eth.BlockID `json:"l1"`
}

// lokahiStatus is lokahi's answer to interop_chainStatus: its heads are nil until it caught up
// with its sources.
type lokahiStatus struct {
	Epoch     common.Hash   `json:"epoch"`
	Verified  *verifierHead `json:"verified"`
	Finalized *verifierHead `json:"finalized"`
	Denied    []eth.BlockID `json:"denied"`
}

// chainStatus asks lokahi for chainID's status, with every denied block.
func chainStatus(ctx context.Context, client *rpc.Client, chainID uint64) (lokahiStatus, error) {
	var status lokahiStatus
	err := client.CallContext(ctx, &status, "interop_chainStatus", chainID, 0)
	return status, err
}

// errNotCaughtUp is lokahi answering no heads, as until it caught up with its sources.
var errNotCaughtUp = errors.New("lokahi answers no heads yet")

// verifiedHead asks lokahi for chainID's verified head.
func verifiedHead(ctx context.Context, client *rpc.Client, chainID uint64) (verifierHead, error) {
	status, err := chainStatus(ctx, client, chainID)
	if err != nil {
		return verifierHead{}, err
	}
	if status.Verified == nil {
		return verifierHead{}, errNotCaughtUp
	}
	return *status.Verified, nil
}

// verifiedHeads asks lokahi for each node's chain's verified head, waiting until lokahi caught up
// with its sources and answers one.
func verifiedHeads(t devtest.T, lokahi *sysgo.Lokahi, nodes []*dsl.L2CLNode) map[*dsl.L2CLNode]verifierHead {
	ctx, cancel := context.WithTimeout(t.Ctx(), time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	heads := make(map[*dsl.L2CLNode]verifierHead, len(nodes))
	for _, node := range nodes {
		chainID, ok := node.ChainID().Uint64()
		t.Require().True(ok, "chain ID fits in 64 bits")
		var head verifierHead
		var lastErr error
		err := wait.For(ctx, 50*time.Millisecond, func() (bool, error) {
			head, lastErr = verifiedHead(ctx, client, chainID)
			return lastErr == nil, nil
		})
		t.Require().NoError(err, "interop_chainStatus: %v", lastErr)
		heads[node] = head
	}
	return heads
}

// verifiedAt reports whether lokahi verified timestamp, also while it answers no heads because it
// has not caught up with its sources: superroot_atTimestamp answers verified timestamps from its
// store.
func verifiedAt(ctx context.Context, client *rpc.Client, timestamp uint64) (bool, error) {
	var resp struct {
		Data *json.RawMessage `json:"data"`
	}
	if err := client.CallContext(ctx, &resp, "superroot_atTimestamp", hexutil.Uint64(timestamp)); err != nil {
		return false, err
	}
	return resp.Data != nil, nil
}

// safeHeadWatch polls nodes' safe heads until stopped and records the first time one decreases.
type safeHeadWatch struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	err    error
}

func watchSafeHeads(t devtest.T, nodes []*dsl.L2CLNode) *safeHeadWatch {
	return watchSafeHeadsDropping(t, nodes, 0)
}

// watchSafeHeadsDropping is [watchSafeHeads] that lets each node's safe head drop to its
// finalized head up to drops times, as a node does when lokahi's epoch changes.
func watchSafeHeadsDropping(t devtest.T, nodes []*dsl.L2CLNode, drops int) *safeHeadWatch {
	ctx, cancel := context.WithCancel(t.Ctx())
	w := &safeHeadWatch{cancel: cancel}
	for _, node := range nodes {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			var last uint64
			dropped := 0
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
					t.Logger().Warn("Safe head moved back", "node", node.String(), "from", last, "to", safe,
						"finalized", status.FinalizedL2.Number)
					if dropped < drops && status.SafeL2.ID() == status.FinalizedL2.ID() {
						dropped++
					} else {
						w.fail(fmt.Errorf("%s: safe head moved back from %d to %d", node, last, safe))
					}
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
