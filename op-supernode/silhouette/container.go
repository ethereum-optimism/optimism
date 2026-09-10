package silhouette

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// ErrNoExecutionData is returned by surfaces that would require re-executing a silhouette chain.
var ErrNoExecutionData = errors.New("proof-carried chain has no execution data")

// Container adds proof-carried message ingestion to a stock verifier chain container.
// Safety labels, rewinds, and derivation remain the stock container's responsibility.
type Container struct {
	cc.InteropChain
	log    log.Logger
	source InteropSource
	sink   func() *LogSink
}

var (
	_ cc.InteropChain         = (*Container)(nil)
	_ cc.MessageIngestion     = (*Container)(nil)
	_ cc.ProvenMessageImports = (*Container)(nil)
)

// NewContainer wraps a verifier chain container. There is deliberately no sequencing posture.
func NewContainer(logger log.Logger, inner cc.InteropChain, source InteropSource, sink func() *LogSink) *Container {
	return &Container{InteropChain: inner, log: logger, source: source, sink: sink}
}

// IngestionSource declares that initiating messages are carried by verified proof batches.
func (c *Container) IngestionSource() cc.IngestionSource { return cc.IngestionProven }

// FetchReceipts refuses: proof batches declare exported messages without exposing private receipts.
func (c *Container) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	return nil, nil, fmt.Errorf("chain %s: FetchReceipts for block %s: %w", c.ID(), blockID, ErrNoExecutionData)
}

// ProvenExecMsgs returns the executing-message set committed by the proof wire. Map keys are set
// ordinals, not private receipt log indices; same-timestamp imports are rejected by the wire codec.
func (c *Container) ProvenExecMsgs(blockNum uint64) (map[uint32]*messages.ExecutingMessage, bool, error) {
	lookupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	fact, err := c.source.InteropBlock(lookupCtx, blockNum)
	cancel()
	if err != nil {
		return nil, false, fmt.Errorf("chain %s: read interop facts for block %d: %w", c.ID(), blockNum, err)
	}
	if fact == nil {
		return nil, false, fmt.Errorf("chain %s: no facts for block %d, so this node cannot say what it "+
			"imported: %w", c.ID(), blockNum, ErrNoExecutionData)
	}
	if fact.Denied {
		// The proof table deliberately retains the denied fact until a corrected proof supersedes
		// it. Once the canonical EL has a different hash at this height, however, that block is the
		// stock deposits-only replacement and therefore has a known-empty import set.
		lookupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		env, payloadErr := c.InteropChain.PayloadByNumber(lookupCtx, blockNum)
		cancel()
		if payloadErr != nil {
			return nil, false, fmt.Errorf("read canonical replacement at block %d: %w", blockNum, payloadErr)
		}
		if env != nil && env.ExecutionPayload != nil && env.ExecutionPayload.BlockHash != fact.Hash {
			return map[uint32]*messages.ExecutingMessage{}, true, nil
		}
	}
	if !fact.ExecMsgsKnown {
		return nil, false, nil
	}
	msgs := make(map[uint32]*messages.ExecutingMessage, len(fact.ExecMsgs))
	for i := range fact.ExecMsgs {
		msg := fact.ExecMsgs[i]
		msgs[uint32(i)] = &msg //nolint:gosec // bounded by a block's gas limit
	}
	return msgs, true, nil
}

// InvalidateBlock records the proof verdict and delegates the rewind to the stock chain container.
// That stops and restarts the unmodified op-node virtual node at the invalid block's parent. The
// replacement itself still comes from the private LightCL sequencer and its corrected proof.
func (c *Container) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32, parentPayload *eth.ExecutionPayloadEnvelope) (bool, error) {
	rewound, rewindErr := c.InteropChain.InvalidateBlock(ctx, height, payloadHash, decisionTimestamp,
		stateRoot, messagePasserStorageRoot, parentPayload)
	markErr := c.source.MarkDenied(ctx, height, payloadHash)
	if rewindErr != nil {
		return rewound, rewindErr
	}
	if markErr != nil {
		return rewound, markErr
	}
	if err := c.sealReplacement(ctx, height, payloadHash); err != nil {
		return rewound, err
	}
	c.log.Warn("invalidated proof block and completed stock verifier rewind",
		"chain", c.ID(), "height", height, "hash", payloadHash,
		"decisionTimestamp", decisionTimestamp)
	return rewound, nil
}

const replacementSealTimeout = 30 * time.Second

// sealReplacement waits for the unmodified verifier to derive the stock deposits-only replacement
// through the Silhouette EL, then gives the interop LogsDB the same canonical-block update a normal
// receipts ingester would make. Until this happens the DB still names the invalid proof block, so
// the next verification round would mistake the replacement itself for a hash conflict.
func (c *Container) sealReplacement(ctx context.Context, height uint64, deniedHash common.Hash) error {
	if c.sink == nil {
		return nil
	}
	sink := c.sink()
	if sink == nil {
		return errors.New("silhouette replacement log sink is not attached")
	}
	waitCtx, cancel := context.WithTimeout(ctx, replacementSealTimeout)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		env, err := c.InteropChain.PayloadByNumber(waitCtx, height)
		if err == nil && env != nil && env.ExecutionPayload != nil {
			payload := env.ExecutionPayload
			if payload.BlockHash != deniedHash {
				fact, factErr := c.source.InteropBlock(waitCtx, height)
				if factErr != nil {
					return fmt.Errorf("read replacement facts at block %d: %w", height, factErr)
				}
				if fact != nil && fact.Hash == payload.BlockHash && !fact.Replacement {
					// A corrected proof won the race and its acceptance already sealed this block.
					return nil
				}
				parent := eth.BlockID{Hash: payload.ParentHash, Number: height - 1}
				block := eth.BlockID{Hash: payload.BlockHash, Number: height}
				if err := sink.ReplaceWithEmpty(parent, block, uint64(payload.Timestamp)); err != nil {
					return fmt.Errorf("record deposits-only replacement %s: %w", block, err)
				}
				return nil
			}
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wait for deposits-only replacement at block %d: %w", height, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Container) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	removed, err := c.InteropChain.PruneDeniedAtOrAfterTimestamp(timestamp)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.source.PruneDenied(ctx, removed); err != nil {
		return nil, err
	}
	return removed, nil
}
