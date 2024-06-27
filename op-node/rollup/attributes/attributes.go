package attributes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Engine interface {
	derive.EngineControl

	SetUnsafeHead(eth.L2BlockRef)
	SetSafeHead(eth.L2BlockRef)
	SetBackupUnsafeL2Head(block eth.L2BlockRef, triggerReorg bool)
	SetPendingSafeL2Head(eth.L2BlockRef)

	PendingSafeL2Head() eth.L2BlockRef
	BackupUnsafeL2Head() eth.L2BlockRef
}

type L2 interface {
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
}

type AttributesHandler struct {
	log log.Logger
	cfg *rollup.Config

	ec Engine
	l2 L2

	attributes *derive.AttributesWithParent
}

func NewAttributesHandler(log log.Logger, cfg *rollup.Config, ec Engine, l2 L2) *AttributesHandler {
	return &AttributesHandler{
		log:        log,
		cfg:        cfg,
		ec:         ec,
		l2:         l2,
		attributes: nil,
	}
}

func (eq *AttributesHandler) HasAttributes() bool {
	return eq.attributes != nil
}

func (eq *AttributesHandler) SetAttributes(attributes *derive.AttributesWithParent) {
	eq.attributes = attributes
}

// Proceed processes block attributes, if any.
// Proceed returns io.EOF if there are no attributes to process.
// Proceed returns a temporary, reset, or critical error like other derivers.
// Proceed returns no error if the safe-head may have changed.
func (eq *AttributesHandler) Proceed(ctx context.Context) error {
	if eq.attributes == nil {
		return io.EOF
	}
	// validate the safe attributes before processing them. The engine may have completed processing them through other means.
	if eq.ec.PendingSafeL2Head() != eq.attributes.Parent {
		// Previously the attribute's parent was the pending safe head. If the pending safe head advances so pending safe head's parent is the same as the
		// attribute's parent then we need to cancel the attributes.
		if eq.ec.PendingSafeL2Head().ParentHash == eq.attributes.Parent.Hash {
			eq.log.Warn("queued safe attributes are stale, safehead progressed",
				"pending_safe_head", eq.ec.PendingSafeL2Head(), "pending_safe_head_parent", eq.ec.PendingSafeL2Head().ParentID(),
				"attributes_parent", eq.attributes.Parent)
			eq.attributes = nil
			return nil
		}
		// If something other than a simple advance occurred, perform a full reset
		return derive.NewResetError(fmt.Errorf("pending safe head changed to %s with parent %s, conflicting with queued safe attributes on top of %s",
			eq.ec.PendingSafeL2Head(), eq.ec.PendingSafeL2Head().ParentID(), eq.attributes.Parent))
	}
	if eq.ec.PendingSafeL2Head().Number < eq.ec.UnsafeL2Head().Number {
		if err := eq.consolidateNextSafeAttributes(ctx, eq.attributes); err != nil {
			return err
		}
		eq.attributes = nil
		return nil
	} else if eq.ec.PendingSafeL2Head().Number == eq.ec.UnsafeL2Head().Number {
		if err := eq.forceNextSafeAttributes(ctx, eq.attributes); err != nil {
			return err
		}
		eq.attributes = nil
		return nil
	} else {
		// For some reason the unsafe head is behind the pending safe head. Log it, and correct it.
		eq.log.Error("invalid sync state, unsafe head is behind pending safe head", "unsafe", eq.ec.UnsafeL2Head(), "pending_safe", eq.ec.PendingSafeL2Head())
		eq.ec.SetUnsafeHead(eq.ec.PendingSafeL2Head())
		return nil
	}
}

// consolidateNextSafeAttributes tries to match the next safe attributes against the existing unsafe chain,
// to avoid extra processing or unnecessary unwinding of the chain.
// However, if the attributes do not match, they will be forced with forceNextSafeAttributes.
func (eq *AttributesHandler) consolidateNextSafeAttributes(ctx context.Context, attributes *derive.AttributesWithParent) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	envelope, err := eq.l2.PayloadByNumber(ctx, eq.ec.PendingSafeL2Head().Number+1)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// engine may have restarted, or inconsistent safe head. We need to reset
			return derive.NewResetError(fmt.Errorf("expected engine was synced and had unsafe block to reconcile, but cannot find the block: %w", err))
		}
		return derive.NewTemporaryError(fmt.Errorf("failed to get existing unsafe payload to compare against derived attributes from L1: %w", err))
	}
	if err := AttributesMatchBlock(eq.cfg, attributes.Attributes, eq.ec.PendingSafeL2Head().Hash, envelope, eq.log); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1", "err", err, "unsafe", eq.ec.UnsafeL2Head(), "pending_safe", eq.ec.PendingSafeL2Head(), "safe", eq.ec.SafeL2Head())
		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		return eq.forceNextSafeAttributes(ctx, attributes)
	}
	ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		return derive.NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}
	eq.ec.SetPendingSafeL2Head(ref)
	if attributes.IsLastInSpan {
		eq.ec.SetSafeHead(ref)
	}
	// unsafe head stays the same, we did not reorg the chain.
	return nil
}

// forceNextSafeAttributes inserts the provided attributes, reorging away any conflicting unsafe chain.
func (eq *AttributesHandler) forceNextSafeAttributes(ctx context.Context, attributes *derive.AttributesWithParent) error {
	attrs := attributes.Attributes
	errType, err := eq.ec.StartPayload(ctx, eq.ec.PendingSafeL2Head(), attributes, true)
	if err == nil {
		_, errType, err = eq.ec.ConfirmPayload(ctx, async.NoOpGossiper{}, &conductor.NoOpConductor{})
	}
	if err != nil {
		switch errType {
		case derive.BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			return derive.NewTemporaryError(fmt.Errorf("temporarily cannot insert new safe block: %w", err))
		case derive.BlockInsertPrestateErr:
			_ = eq.ec.CancelPayload(ctx, true)
			return derive.NewResetError(fmt.Errorf("need reset to resolve pre-state problem: %w", err))
		case derive.BlockInsertPayloadErr:
			_ = eq.ec.CancelPayload(ctx, true)
			eq.log.Warn("could not process payload derived from L1 data, dropping batch", "err", err)
			// Count the number of deposits to see if the tx list is deposit only.
			depositCount := 0
			for _, tx := range attrs.Transactions {
				if len(tx) > 0 && tx[0] == types.DepositTxType {
					depositCount += 1
				}
			}
			// Deposit transaction execution errors are suppressed in the execution engine, but if the
			// block is somehow invalid, there is nothing we can do to recover & we should exit.
			if len(attrs.Transactions) == depositCount {
				eq.log.Error("deposit only block was invalid", "parent", attributes.Parent, "err", err)
				return derive.NewCriticalError(fmt.Errorf("failed to process block with only deposit transactions: %w", err))
			}
			// Revert the pending safe head to the safe head.
			eq.ec.SetPendingSafeL2Head(eq.ec.SafeL2Head())
			// suppress the error b/c we want to retry with the next batch from the batch queue
			// If there is no valid batch the node will eventually force a deposit only block. If
			// the deposit only block fails, this will return the critical error above.

			// Try to restore to previous known unsafe chain.
			eq.ec.SetBackupUnsafeL2Head(eq.ec.BackupUnsafeL2Head(), true)

			// drop the payload (by returning no error) without inserting it into the engine
			return nil
		default:
			return derive.NewCriticalError(fmt.Errorf("unknown InsertHeadBlock error type %d: %w", errType, err))
		}
	}
	return nil
}
