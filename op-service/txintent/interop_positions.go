package txintent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Public message positions, for a chain whose receipts are not its public presence.
//
// # The one assumption this file relaxes
//
// InteropOutput.FromReceipt builds a message identifier straight out of a receipt log: the log's
// own emitter is the Origin, the log's own block-level index is the LogIndex. That is correct for
// every ordinary chain, because an ordinary chain's blocks ARE what its counterparties read.
//
// A private-interop chain is the exception, and the only one (op-private-interop/docs/DESIGN.md,
// "Canonical message positions"). Its public presence is a RENDERING: one public block per private
// block, at the same number and the same timestamp, carrying only the logs the export policy makes
// public, re-indexed from zero. Everything that judges a message -- the counterparty's supernode,
// the interop filter, relayers -- reads the rendering and has never seen a private receipt. So a
// message emitted on such a chain is named by its position on the rendering, which differs from its
// receipt position in two coordinates: the log index always, and the emitter address whenever the
// log is re-emitted through a generic replayer rather than at its own predeploy.
//
// # Why the seam is here
//
// FromReceipt is the ONE place in the devstack where an identifier is minted from a receipt. Every
// helper a test uses -- (*EOA).SendInitMessage, SendExecMessage, ExecuteIndexed, RelayIndexed, the
// messenger SendTrigger/RelayTrigger path -- reads the entries it produced. Patching it once means
// tests keep calling the same helpers and never learn the difference, which is the whole point of
// the plug-in thesis (op-private-interop/docs/DESIGN.md, "Testing": the identifier seam).
//
// # And why it costs nothing when nobody registers
//
// With no resolver registered -- every stock preset, every production use -- resolution is one
// atomic load and a return. The entries FromReceipt produces are then byte-identical to what it
// produced before this file existed.

// PublicPosition is where a chain's counterparties see one of its logs.
//
// The zero value means "this log has no public position": it exists on the emitting chain and the
// export policy does not publish it, so no counterparty can execute it. Such a log's identifier is
// left exactly as the stock path built it -- an honest description of a private receipt position,
// which any judge will (correctly) reject.
type PublicPosition struct {
	// Origin is the emitter address the log carries publicly. It differs from the private emitter
	// when the log is republished through a generic replayer contract, which emits at its own
	// address.
	Origin common.Address
	// LogIndex is the log's block-level index in the public block.
	LogIndex uint32
	// Public reports whether this log has a public position at all.
	Public bool
}

// PositionResolver answers, for one chain, where that chain's logs appear publicly.
//
// Implementations live in the devstack (op-devstack/presets), which is where the handles on both
// halves of a pair exist. Nothing in production registers one: a production relayer reads the
// rendering directly, which is the same answer arrived at without a translation step.
type PositionResolver interface {
	// Owns reports whether the given block is one this resolver's chain produced.
	//
	// It exists because a chain ID does not uniquely name a chain in a test process: a private
	// chain and its rendering share one, and a process running several worlds in parallel holds
	// several chains at the same ID. The block is the disambiguator.
	Owns(ctx context.Context, block eth.BlockRef) bool

	// ResolvePositions returns one entry per log of rec, in the receipt's own order.
	//
	// It may block: a log's public position does not exist until the chain's public presence has
	// caught up with the block that emitted it. Implementations bound that wait themselves and
	// return an error rather than hanging.
	ResolvePositions(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef) ([]PublicPosition, error)
}

var (
	positionResolversMu sync.RWMutex
	positionResolvers   = map[eth.ChainID][]PositionResolver{}
	// positionResolverCount is read on the hot path before the mutex is touched, so that a process
	// with no resolvers registered pays one atomic load per receipt and nothing else.
	positionResolverCount atomic.Int64
)

// RegisterPositionResolver makes a chain's logs resolve to their public positions for the rest of
// the process, and returns the function that undoes it.
//
// Registration is by chain ID and is additive rather than exclusive: several worlds in one test
// process legitimately share a chain ID, and Owns picks between them.
func RegisterPositionResolver(chainID eth.ChainID, resolver PositionResolver) (unregister func()) {
	if resolver == nil {
		return func() {}
	}
	positionResolversMu.Lock()
	positionResolvers[chainID] = append(positionResolvers[chainID], resolver)
	positionResolversMu.Unlock()
	positionResolverCount.Add(1)

	var once sync.Once
	return func() {
		once.Do(func() {
			positionResolversMu.Lock()
			kept := positionResolvers[chainID][:0]
			for _, r := range positionResolvers[chainID] {
				if r != resolver {
					kept = append(kept, r)
				}
			}
			if len(kept) == 0 {
				delete(positionResolvers, chainID)
			} else {
				positionResolvers[chainID] = kept
			}
			positionResolversMu.Unlock()
			positionResolverCount.Add(-1)
		})
	}
}

// positionResolverFor returns the resolver that owns the given block, or nil if the chain publishes
// its own receipts (the ordinary case).
func positionResolverFor(ctx context.Context, chainID eth.ChainID, block eth.BlockRef) (PositionResolver, error) {
	if positionResolverCount.Load() == 0 {
		return nil, nil
	}
	positionResolversMu.RLock()
	candidates := append([]PositionResolver(nil), positionResolvers[chainID]...)
	positionResolversMu.RUnlock()

	switch len(candidates) {
	case 0:
		return nil, nil
	case 1:
		return candidates[0], nil
	}
	for _, r := range candidates {
		if r.Owns(ctx, block) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("chain %s has %d registered position resolvers and none of them owns block %s: "+
		"a receipt cannot be given a public position by a chain that did not produce it", chainID, len(candidates), block)
}

// applyPublicPositions rewrites the identifiers of an InteropOutput to the positions that name the
// same logs publicly. It is a no-op when the emitting chain publishes its own receipts.
func applyPublicPositions(ctx context.Context, out *InteropOutput, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error {
	resolver, err := positionResolverFor(ctx, chainID, includedIn)
	if err != nil || resolver == nil {
		return err
	}
	positions, err := resolver.ResolvePositions(ctx, rec, includedIn)
	if err != nil {
		return fmt.Errorf("resolving the public positions of the %d logs of tx %s on chain %s: %w",
			len(rec.Logs), rec.TxHash, chainID, err)
	}
	if len(positions) != len(out.Entries) {
		return fmt.Errorf("the position resolver for chain %s returned %d positions for a receipt with %d logs; "+
			"a resolver must answer for every log, in order, so that a message's index into the entries does not move",
			chainID, len(positions), len(out.Entries))
	}
	for i, pos := range positions {
		if !pos.Public {
			continue
		}
		// Block number and timestamp are NOT touched: a rendering is block-for-block with the chain
		// it renders, so those two coordinates already describe the public block.
		out.Entries[i].Identifier.Origin = pos.Origin
		out.Entries[i].Identifier.LogIndex = pos.LogIndex
	}
	return nil
}
