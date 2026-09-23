package poller

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// The standing rendering invariant, for a private interop pair.
//
// A pair's central claim is block-for-block correspondence: the public rendering carries one block
// per private block, at the same number and the same timestamp. Everything downstream rests on it --
// a message's identity is its position on the rendering, and a position only means anything if the
// two chains agree about where a block number sits in time.
//
// So this runs for the whole of any suite that uses the preset, and it costs nothing when the system
// is healthy. That is why it is on by default with an explicit opt-out (the `WithoutCheck`
// precedent) rather than something a test remembers to switch on.
//
// # What it asserts, and what it deliberately does not
//
// It asserts only things that CANNOT flake:
//
//   - correspondence: at every rendering block the rendering has derived, the private chain's block
//     at that number has the same timestamp. Hashes are NOT compared, and never will be -- the two
//     chains have different content by design, and their genesis hashes differ on purpose;
//   - log equality: the rendering carries exactly the logs selected from the corresponding private
//     block, in the same order and with byte-identical topics and data. Core interop logs retain
//     their emitter; configured extra-emitter logs move to EventReplayer by design;
//   - monotonicity: neither the rendering's safe head nor the private chain's claim-driven safe head
//     ever goes backwards. A regression on the private side is the sharp one: a follow-mode sequencer
//     force-resets onto whatever local-safe ref it is told, so a follow source that reported a lower
//     ref would rewind a chain that had already been publicly committed to. The supernode's follow
//     module serves high-water marks for exactly this reason, and this is that contract, checked;
//   - agreement: the private safe head the follow module serves names a block the private chain
//     actually has, at the hash it actually has. Under snap-to-commitment a mismatch is not a
//     serving fault -- the module has no private EL to compare against and serves claims verbatim --
//     so what this catches is the condition itself: the operator's chain and the operator's own
//     public claims have parted company. In production that is a monitoring alert; in a test it is
//     a failure, because nothing here should be able to produce it.
//
// PROGRESS is not asserted here, and that is a considered omission rather than a gap. "The heads
// advance" is a real requirement, but on a background poller it is a timing assumption, and a
// timing assumption in a standing checker is a flake in every suite it watches. Tests that care --
// and the pair's own e2e does -- call RequireRenderingReached, which is bounded and explicit.
//
// Interop invalidations (testing plan section 4's third assertion) are not scraped yet: the
// supernode's metrics service never reports the port it bound, so reading a counter off it needs the
// grab-and-release trick, and that is worth doing when there is a second consumer for it.

// renderingInvariantPollInterval is how often the check steps. The private chain builds every two
// seconds, so this samples several times per block without being a load on either node.
const renderingInvariantPollInterval = 500 * time.Millisecond

// renderingInvariantMaxBlocksPerPoll bounds one poll's catch-up work, so that a checker starting
// behind a long chain walks it incrementally rather than stalling in one call.
const renderingInvariantMaxBlocksPerPoll = 64

// RenderingInvariant is the running check.
type RenderingInvariant struct {
	t devtest.T

	privateEL   *dsl.L2ELNode
	renderingEL *dsl.L2ELNode
	privateCL   *dsl.L2CLNode
	renderingCL *dsl.L2CLNode
	emitters    render.EmitterSet

	cancel context.CancelFunc
	done   <-chan struct{}

	mu sync.Mutex
	// violations are the invariant breaches seen so far, each already described in full. They are
	// collected rather than fataled on the spot: this runs on a background goroutine, where a
	// t.Fatal would be reported against whatever test happened to be in the foreground.
	violations []string
	// checked is the highest rendering block whose correspondence has been verified.
	checked uint64
	// haveChecked distinguishes "nothing verified yet" from "verified block 0".
	haveChecked bool
	// renderingSafe and privateSafe are the highest heads seen, for the monotonicity check.
	renderingSafe, privateSafe uint64
	// polls and pollErrs count what happened, so a checker that never managed to read anything
	// cannot pass by silence.
	polls, pollErrs int
}

// StartRenderingInvariant begins checking a private interop pair in the background.
//
// It registers a t.Cleanup that stops the poller and then fails the test if the invariant was ever
// broken, so a caller never manages its lifecycle and never has to remember to assert.
func StartRenderingInvariant(
	privateEL, renderingEL *dsl.L2ELNode,
	privateCL, renderingCL *dsl.L2CLNode,
	emitters render.EmitterSet,
) *RenderingInvariant {
	t := privateEL.Escape().T()
	ctx, cancel := context.WithCancel(t.Ctx())
	done := make(chan struct{})

	r := &RenderingInvariant{
		t:           t,
		privateEL:   privateEL,
		renderingEL: renderingEL,
		privateCL:   privateCL,
		renderingCL: renderingCL,
		emitters:    emitters,
		cancel:      cancel,
		done:        done,
	}
	t.Cleanup(func() {
		r.stop()
		r.RequireNoViolations()
	})

	go r.run(ctx, done)
	return r
}

func (r *RenderingInvariant) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(renderingInvariantPollInterval)
	defer ticker.Stop()
	for {
		r.poll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// poll performs one step. Every RPC failure is a counted, silent skip: a node that is briefly
// unreachable is a devstack fact of life, and it is not what this check is about.
func (r *RenderingInvariant) poll(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	r.mu.Lock()
	r.polls++
	r.mu.Unlock()

	renderingStatus, err := r.renderingCL.Escape().RollupAPI().SyncStatus(ctx)
	if err != nil {
		r.countErr()
		return
	}
	r.observeMonotone("the rendering's safe head", renderingStatus.SafeL2.Number, &r.renderingSafe)

	privateStatus, err := r.privateCL.Escape().RollupAPI().SyncStatus(ctx)
	if err != nil {
		r.countErr()
		return
	}
	r.observeMonotone("the private chain's claim-driven safe head", privateStatus.SafeL2.Number, &r.privateSafe)
	r.checkClaimAgreement(ctx, privateStatus.SafeL2)

	r.checkCorrespondence(ctx, renderingStatus.SafeL2.Number)
}

// checkCorrespondence walks the rendering blocks that have become safe since the last poll and
// checks each against the private chain's block at the same number.
//
// It walks only up to the rendering's SAFE head, not its unsafe one. A rendering block is derived
// from an L1 batch, and reading below the safe head means L1 reorg handling is the rendering node's
// problem rather than this checker's -- the same reason the follow module scans there and nowhere
// else.
func (r *RenderingInvariant) checkCorrespondence(ctx context.Context, renderingSafe uint64) {
	r.mu.Lock()
	next := r.checked + 1
	if !r.haveChecked {
		next = 0
	}
	r.mu.Unlock()

	last := renderingSafe
	if last > next+renderingInvariantMaxBlocksPerPoll {
		last = next + renderingInvariantMaxBlocksPerPoll
	}
	for num := next; num <= last; num++ {
		rendered, err := r.renderingEL.Escape().L2EthClient().BlockRefByNumber(ctx, num)
		if err != nil {
			r.countErr()
			return
		}
		private, err := r.privateEL.Escape().L2EthClient().BlockRefByNumber(ctx, num)
		if err != nil {
			// The private chain not having block `num` yet is not a violation on its own -- it means
			// the rendering has run ahead, which cannot happen, or that the node is briefly
			// unreachable, which can. Neither is worth a false failure; the next poll retries.
			r.countErr()
			return
		}
		if rendered.Time != private.Time {
			r.violation(fmt.Sprintf(
				"block-for-block correspondence broken at block %d: the rendering is at timestamp %d, the private chain at %d. "+
					"The rendering synthesizes one public block per private block at the same number AND the same timestamp; "+
					"a mismatch means every message identity at or after this height names a position that does not describe the private chain",
				num, rendered.Time, private.Time))
		}
		if err := r.checkLogEquality(ctx, num, private.Hash, rendered.Hash); err != nil {
			r.countErr()
			return
		}
		r.mu.Lock()
		r.checked, r.haveChecked = num, true
		r.mu.Unlock()
	}
}

// checkLogEquality asserts the pair's central content invariant. It deliberately reads receipts
// from both execution engines rather than predicting rendering execution: a reverted replay is
// exactly the failure this check exists to expose.
func (r *RenderingInvariant) checkLogEquality(
	ctx context.Context,
	number uint64,
	privateHash common.Hash,
	renderingHash common.Hash,
) error {
	privateLogs, err := blockLogs(ctx, r.privateEL, privateHash)
	if err != nil {
		return err
	}
	publicLogs, err := blockLogs(ctx, r.renderingEL, renderingHash)
	if err != nil {
		return err
	}
	expected := render.RenderedLogs(privateLogs, r.emitters)
	if len(publicLogs) != len(expected) {
		r.violation(fmt.Sprintf(
			"rendering log equality broken at block %d: selected %d private logs but derived %d rendering logs; a missing or extra log renumbers every later message in the block",
			number, len(expected), len(publicLogs)))
		return nil
	}
	for i, want := range expected {
		got := publicLogs[i]
		wantAddress := want.Log.Address
		if wantAddress != predeploys.L2toL2CrossDomainMessengerAddr && wantAddress != predeploys.CrossL2InboxAddr {
			wantAddress = predeploys.EventReplayerAddr
		}
		if got.Index != uint(i) || got.Address != wantAddress ||
			!slices.Equal(got.Topics, want.Log.Topics) || !bytes.Equal(got.Data, want.Log.Data) {
			r.violation(fmt.Sprintf(
				"rendering log equality broken at block %d index %d: got emitter=%s blockIndex=%d topics=%v data=%x; want emitter=%s blockIndex=%d topics=%v data=%x",
				number, i, got.Address, got.Index, got.Topics, got.Data,
				wantAddress, i, want.Log.Topics, want.Log.Data))
			return nil
		}
	}
	return nil
}

func blockLogs(ctx context.Context, el *dsl.L2ELNode, blockHash common.Hash) ([]*types.Log, error) {
	_, receipts, err := el.Escape().L2EthClient().FetchReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	var out []*types.Log
	for _, receipt := range receipts.Geth() {
		out = append(out, receipt.Logs...)
	}
	return out, nil
}

// checkClaimAgreement verifies that the safe head the follow module served names a block the private
// chain actually has, at the hash it actually has.
//
// It is the divergence alarm, from outside. The module computes its refs from the CLAIM and has no
// private EL to check them against; under snap-to-commitment that is deliberate, because a diverged
// sequencer force-resetting onto the publicly claimed chain is recovery TO the operator's binding
// statement rather than away from it. Nothing on this side of the wire is therefore supposed to
// disagree, and a disagreement means the operator's chain and the operator's own claims have parted
// company -- a monitoring alert in production, and here a failure, since a devstack pair has no way
// to produce one legitimately.
func (r *RenderingInvariant) checkClaimAgreement(ctx context.Context, safe eth.L2BlockRef) {
	if safe == (eth.L2BlockRef{}) {
		return
	}
	local, err := r.privateEL.Escape().L2EthClient().BlockRefByNumber(ctx, safe.Number)
	if err != nil {
		r.countErr()
		return
	}
	if local.Hash != safe.Hash {
		r.violation(fmt.Sprintf(
			"the private chain's served safe head at block %d is %s, but the private chain has %s. "+
				"The supernode's follow module serves the CLAIM verbatim, so this is the private chain and its own public "+
				"claims having parted company -- and a sequencing follower force-resets onto the ref it is served",
			safe.Number, safe.Hash, local.Hash))
	}
}

// observeMonotone records a head and flags any regression.
func (r *RenderingInvariant) observeMonotone(what string, seen uint64, high *uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if seen < *high {
		r.violations = append(r.violations, fmt.Sprintf(
			"%s went backwards, from block %d to block %d. A safe head is a public commitment; retracting one contradicts something already published",
			what, *high, seen))
		return
	}
	*high = seen
}

func (r *RenderingInvariant) violation(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.violations = append(r.violations, msg)
}

func (r *RenderingInvariant) countErr() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pollErrs++
}

// RequireNoViolations fails the test if the invariant was ever broken. The cleanup registered by
// StartRenderingInvariant calls it, so a test only calls it directly to check early.
func (r *RenderingInvariant) RequireNoViolations() {
	r.mu.Lock()
	violations := append([]string(nil), r.violations...)
	r.mu.Unlock()
	r.t.Require().Emptyf(violations, "the private interop pair broke its standing invariant:\n%v", violations)
}

// RenderingSafeHead is the highest rendering safe block the checker has seen.
func (r *RenderingInvariant) RenderingSafeHead() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.renderingSafe
}

// PrivateSafeHead is the highest claim-driven private safe block the checker has seen.
func (r *RenderingInvariant) PrivateSafeHead() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.privateSafe
}

// CheckedThrough is the highest rendering block whose correspondence has been verified. It is the
// honest measure of how much this checker has actually done, which a passing run should report
// rather than assume.
func (r *RenderingInvariant) CheckedThrough() (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.checked, r.haveChecked
}

// RequireRenderingReached waits until the rendering's safe head has reached a given block.
//
// Progress lives here rather than in the standing poll on purpose: it is the one property of a pair
// that a background checker cannot assert without inventing a deadline, and a deadline invented once
// for every suite is a flake. A test that cares states its own budget.
//
// It is stated as "reached block N" rather than "advanced by N", because both callers know the block
// they are waiting for -- the one carrying their message, or simply the first few -- and a delta
// measured from whenever the call happened to be made is a race with the pair's own progress.
func (r *RenderingInvariant) RequireRenderingReached(block uint64, timeout time.Duration) {
	start := r.RenderingSafeHead()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.RenderingSafeHead() >= block {
			return
		}
		time.Sleep(renderingInvariantPollInterval)
	}
	r.t.Require().Failf("the rendering did not advance",
		"the rendering's safe head was at block %d and did not reach block %d within %s. "+
			"The rendering only advances when a range's batch lands on L1 and is derived, so a stall here is the builder or the batch inbox -- "+
			"the checker made %d polls with %d read failures",
		start, block, timeout, r.polls, r.pollErrs)
}

// RequirePrivateSafeReached waits until the private chain's CLAIM-DRIVEN safe head has reached a
// given block.
//
// It is a strictly later event than rendering progress and worth waiting for separately: the
// rendering advancing only means a batch landed and was derived, while this means the follower also
// found the range's claim in that batch, resolved its terminal block against the private EL, and
// matched the two. A test that gets the first and not the second has a working publisher and a
// broken safety loop, and the two failures deserve different messages.
func (r *RenderingInvariant) RequirePrivateSafeReached(block uint64, timeout time.Duration) {
	start := r.PrivateSafeHead()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.PrivateSafeHead() >= block {
			return
		}
		time.Sleep(renderingInvariantPollInterval)
	}
	r.t.Require().Failf("the private chain's claim-driven safe head did not advance",
		"the private chain's safe head was at block %d and did not reach block %d within %s, "+
			"while the rendering's safe head reached block %d. "+
			"A private range is safe once its claim has landed in an L1 batch the rendering derived AND the local chain hashes the same at that height, "+
			"so a stall here with a live rendering is the claim -- posted, found, or matched",
		start, block, timeout, r.RenderingSafeHead())
}

func (r *RenderingInvariant) stop() {
	r.cancel()
	<-r.done
}
