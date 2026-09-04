package presets

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// The identifier resolver: the one central patch that lets a stock interop test run against a
// private-interop pair without knowing it.
//
// # What it resolves, and why nothing else has to
//
// A message emitted on the private chain is named, everywhere that matters, by its position on the
// RENDERING (op-private-interop/docs/DESIGN.md, "Canonical message positions"). Two coordinates
// move:
//
//   - the LOG INDEX, always. The private block carries every log it produced; the rendering carries
//     only the export policy's, re-indexed from zero. render.RenderedLogs is the normative
//     definition of which and in what order, and this file imports it rather than restating it --
//     a second implementation of that predicate is a consensus divergence waiting for a block that
//     interleaves;
//   - the ORIGIN, for a log the rendering republishes through the generic EventReplayer rather than
//     at its own predeploy address. The replayer emits at ITS OWN address, so an extra emitter's
//     log is publicly a log of the replayer. Messenger exports and inbox imports keep their
//     addresses, which is the entire reason they are rendered at their predeploys.
//
// Block number and timestamp do NOT move, because the rendering is block-for-block with the chain
// it renders. The standing invariant checker asserts that continuously.
//
// # Read the rendering, do not predict it
//
// The resolver computes the rendered INDEX from the private block through render.RenderedLogs, and
// then reads the log the rendering actually carries at that index -- taking Origin from the real
// public log and cross-checking its payload hash against the private one. That is deliberately
// belt-and-braces: the prediction is what a relayer would compute, the reading is what a judge will
// see, and a test that resolves a position the two disagree about should fail here, in a message
// about the rendering transformation, rather than three minutes later as an unexplained block
// replacement on the counterparty.
//
// It is also why the resolver WAITS. A message has no public position until the rendering has
// derived the block that carries it, and the rendering only advances when a range's batch lands on
// L1. The wait lives here, once, keyed off the preset -- not sprinkled through the tests, which is
// the whole point of the seam.

// privateInteropPositionTimeout bounds the wait for the rendering to derive a message's block.
//
// It has to cover a full range: the builder finishing the range, committing to its private input,
// assembling the claim, the batch landing on L1 and the rendering deriving it. At the devstack's
// default cadence that is seconds; at a large cadence it is the cadence. Five minutes is several
// times the worst case either way, and a resolver that has waited five minutes is not slow, it is
// broken -- which is what the error says.
const privateInteropPositionTimeout = 5 * time.Minute

// privateInteropPositionPoll is how often the wait re-reads the rendering's safe head.
const privateInteropPositionPoll = 250 * time.Millisecond

// privateInteropResolver answers where the private chain's logs appear on its rendering.
type privateInteropResolver struct {
	// privateEL is where the receipt came from: the source of the block's COMPLETE log sequence,
	// which is what the rendering transformation is defined over. A receipt's own logs are not
	// enough -- the rendered index counts across the whole block.
	privateEL *dsl.L2ELNode
	// renderingEL and renderingCL are the public half: the log sequence to read positions out of,
	// and the safe head to wait on.
	renderingEL *dsl.L2ELNode
	renderingCL *dsl.L2CLNode

	// emitters is the export policy. It must be the SAME set the builder was configured with, or
	// the two sides disagree about which logs exist publicly and every index after the first
	// disagreement is wrong.
	emitters render.EmitterSet

	timeout time.Duration
}

var _ txintent.PositionResolver = (*privateInteropResolver)(nil)

// Owns reports whether the block is one the PRIVATE chain produced.
//
// The private chain and its rendering share a chain ID, and a test process may hold several worlds
// at that ID, so the chain ID alone cannot say whose receipt this is. The block hash can: it is the
// one thing the two halves of a pair are guaranteed to disagree about.
func (r *privateInteropResolver) Owns(ctx context.Context, block eth.BlockRef) bool {
	got, err := r.privateEL.Escape().L2EthClient().BlockRefByNumber(ctx, block.Number)
	return err == nil && got.Hash == block.Hash
}

func (r *privateInteropResolver) ResolvePositions(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef) ([]txintent.PublicPosition, error) {
	out := make([]txintent.PublicPosition, len(rec.Logs))

	// The rendered index of each of this receipt's logs, predicted from the private block.
	renderedIndex, err := r.renderedIndexes(ctx, includedIn, rec)
	if err != nil {
		return nil, err
	}
	if len(renderedIndex) == 0 {
		// Nothing this transaction emitted is public. Common and correct: a relay's RelayedMessage
		// is bookkeeping, not a claim, and the export policy leaves it on the private chain.
		return out, nil
	}

	// The rendering has to have derived the block before any of those indexes names anything.
	if err := r.awaitRendering(ctx, includedIn.Number); err != nil {
		return nil, err
	}
	publicLogs, err := r.renderingLogs(ctx, includedIn.Number)
	if err != nil {
		return nil, err
	}

	for i, l := range rec.Logs {
		k, ok := renderedIndex[l.Index]
		if !ok {
			continue
		}
		if int(k) >= len(publicLogs) {
			return nil, fmt.Errorf("log %d of block %d renders at public index %d, but the rendering's block carries %d logs: "+
				"the rendering's log sequence must equal render.RenderedLogs of the private block exactly",
				l.Index, includedIn.Number, k, len(publicLogs))
		}
		public := publicLogs[k]
		if uint32(public.Index) != k {
			return nil, fmt.Errorf("the rendering's block %d carries its %d-th log at block-level index %d: "+
				"a rendering block's logs are indexed 0..k-1 by construction", includedIn.Number, k, public.Index)
		}
		if got, want := payloadHash(public), payloadHash(l); got != want {
			return nil, fmt.Errorf("the rendering's log at index %d in block %d hashes %s, the private log %d hashes %s: "+
				"a replayed log must be byte-identical in topics and data, or the message a counterparty executes is not the message that was sent",
				k, includedIn.Number, got, l.Index, want)
		}
		out[i] = txintent.PublicPosition{
			// From the rendering's own log, not from the private one: an extra emitter's log is
			// republished by the EventReplayer at the replayer's address, and the identifier has to
			// name the emitter a judge will see.
			Origin:   public.Address,
			LogIndex: k,
			Public:   true,
		}
	}
	return out, nil
}

// renderedIndexes maps the block-level index of each of the receipt's PUBLIC logs to its index on
// the rendering. Logs the export policy does not publish are simply absent.
func (r *privateInteropResolver) renderedIndexes(ctx context.Context, block eth.BlockRef, rec *types.Receipt) (map[uint]uint32, error) {
	if len(rec.Logs) == 0 {
		return nil, nil
	}
	logs, err := blockLogs(ctx, r.privateEL, block.Hash)
	if err != nil {
		return nil, fmt.Errorf("reading the private block %d (%s) whose logs define the rendering: %w", block.Number, block.Hash, err)
	}
	rendered := render.RenderedLogs(logs, r.emitters)

	byPrivateIndex := make(map[uint32]uint32, len(rendered))
	for _, rl := range rendered {
		byPrivateIndex[rl.PrivateLogIndex] = rl.RenderedLogIndex
	}
	out := make(map[uint]uint32, len(rec.Logs))
	for _, l := range rec.Logs {
		if k, ok := byPrivateIndex[uint32(l.Index)]; ok {
			out[l.Index] = k
		}
	}
	return out, nil
}

// renderingLogs is the rendering block's complete log sequence at a height.
func (r *privateInteropResolver) renderingLogs(ctx context.Context, number uint64) ([]*types.Log, error) {
	ref, err := r.renderingEL.Escape().L2EthClient().BlockRefByNumber(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("reading the rendering's block %d: %w", number, err)
	}
	logs, err := blockLogs(ctx, r.renderingEL, ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("reading the rendering's block %d (%s): %w", number, ref.Hash, err)
	}
	return logs, nil
}

// awaitRendering blocks until the rendering's safe head has reached a height, or the budget runs
// out.
//
// The rendering's SAFE head, not its unsafe one: the rendering has no sequencer and every block it
// has is derived from an L1 batch, so its safe head is simply how far the pipeline has got. It is
// also the head the supernode's message database follows, which is what will judge the executing
// message this position is about to be used in.
func (r *privateInteropResolver) awaitRendering(ctx context.Context, number uint64) error {
	deadline := time.Now().Add(r.timeout)
	var last uint64
	for {
		status, err := r.renderingCL.Escape().RollupAPI().SyncStatus(ctx)
		if err == nil {
			last = status.SafeL2.Number
			if last >= number {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the rendering's safe head was at block %d and did not reach block %d within %s, "+
				"so the message emitted there has no public position yet. The rendering advances one range at a time: "+
				"a stall here is the builder, the batch on L1, or a cadence larger than this budget",
				last, number, r.timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(privateInteropPositionPoll):
		}
	}
}

// blockLogs returns a block's complete log sequence in block order.
//
// The rendering transformation is defined over the whole block, so this is the shape every caller
// needs; a single receipt's logs would give a rendered index that counts only that transaction.
func blockLogs(ctx context.Context, el *dsl.L2ELNode, blockHash common.Hash) ([]*types.Log, error) {
	_, receipts, err := el.Escape().L2EthClient().FetchReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	var logs []*types.Log
	for _, rec := range receipts.Geth() {
		logs = append(logs, rec.Logs...)
	}
	return logs, nil
}

func payloadHash(l *types.Log) eth.Bytes32 {
	return eth.Bytes32(crypto.Keccak256Hash(messages.LogToMessagePayload(l)))
}
