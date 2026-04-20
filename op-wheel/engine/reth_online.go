package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

const (
	fcuRetryDelay  = 500 * time.Millisecond
	maxFCUAttempts = 20
)

var (
	ErrRewindOverFinalizedHead        = errors.New("cannot rewind past finalized head")
	ErrRewindSyntheticPayloadRejected = errors.New("synthetic payload rejected by engine")
	ErrRewindFCURejected              = errors.New("forkchoice update rejected by engine")
	ErrRewindFCUHeadMismatch          = errors.New("FCU head did not converge to expected value")
)

// RethRewindOnline rewinds a running reth node via the Engine API, without
// touching its database directly. It ports the approach from
// op-supernode/supernode/chain_container/engine_controller/rewind.go:
//
//  1. fetch the target block and build a "synthetic" payload on the same parent
//     by perturbing ExtraData (a header field with no EVM opcode, so the state
//     root is unaffected),
//  2. submit it via NewPayload and FCU -> synthetic to force a reorg off the
//     current canonical chain,
//  3. FCU -> target to restore it as head, with safe/finalized clamped so they
//     never move forward.
//
// This is the online counterpart to RethRewind, which shells out to
// `reth stage unwind` against a stopped node.
func RethRewindOnline(ctx context.Context, lgr log.Logger, engineClient *sources.EngineAPIClient, open client.RPC, to uint64) error {
	targetEnv, err := getPayloadEnvByNumber(ctx, open, to)
	if err != nil {
		return fmt.Errorf("failed to get payload for block %d: %w", to, err)
	}
	target := targetEnv.ExecutionPayload
	parentHash := target.ParentHash

	_, safe, finalized, err := headSafeFinalized(ctx, open)
	if err != nil {
		return fmt.Errorf("failed to get current heads: %w", err)
	}
	if finalized != nil && to < bigs.Uint64Strict(finalized.Number) {
		return fmt.Errorf("%w: target=%d finalized=%d", ErrRewindOverFinalizedHead, to, bigs.Uint64Strict(finalized.Number))
	}

	// Clamp safe/finalized so they don't move forward.
	toSafe, toFinalized := target.BlockHash, target.BlockHash
	if safe != nil && bigs.Uint64Strict(safe.Number) < to {
		toSafe = safe.Hash()
	}
	if finalized != nil && bigs.Uint64Strict(finalized.Number) < to {
		toFinalized = finalized.Hash()
	}

	lgr.Info("Rewinding reth online",
		"target", target.BlockHash, "number", to,
		"parent", parentHash, "safe", toSafe, "finalized", toFinalized)

	syntheticHash, err := insertSyntheticPayload(ctx, lgr, engineClient, targetEnv)
	if err != nil {
		return err
	}

	// FCU to synthetic: parentHash is always in canonical chain of both the
	// original and the synthetic branch, so it's a safe choice for safe/finalized.
	if err := forkchoiceUpdateWithRetry(ctx, lgr, engineClient, open, syntheticHash, parentHash, parentHash); err != nil {
		return fmt.Errorf("FCU to synthetic block failed: %w", err)
	}
	lgr.Info("FCU to synthetic block succeeded", "head", syntheticHash)

	if err := forkchoiceUpdateWithRetry(ctx, lgr, engineClient, open, target.BlockHash, toSafe, toFinalized); err != nil {
		return fmt.Errorf("FCU to target block failed: %w", err)
	}
	lgr.Info("FCU to target block succeeded", "head", target.BlockHash, "safe", toSafe, "finalized", toFinalized)
	return nil
}

// getPayloadEnvByNumber fetches a block as an ExecutionPayloadEnvelope over the open RPC.
func getPayloadEnvByNumber(ctx context.Context, open client.RPC, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	var block *sources.RPCBlock
	if err := open.CallContext(ctx, &block, methodEthGetBlockByNumber, hexutil.Uint64(number).String(), true); err != nil {
		return nil, err
	}
	if block == nil {
		return nil, ethereum.NotFound
	}
	return block.ExecutionPayloadEnvelope(false)
}

// insertSyntheticPayload builds a synthetic envelope from the target and submits it via NewPayload.
// Returns the hash of the synthetic block on success.
func insertSyntheticPayload(ctx context.Context, lgr log.Logger, engineClient *sources.EngineAPIClient, envelope *eth.ExecutionPayloadEnvelope) (common.Hash, error) {
	syntheticEnv := buildSyntheticPayload(envelope)
	syntheticPayload := syntheticEnv.ExecutionPayload

	lgr.Info("Inserting synthetic payload",
		"number", uint64(syntheticPayload.BlockNumber),
		"parent", syntheticPayload.ParentHash, "hash", syntheticPayload.BlockHash)

	status, err := engineClient.NewPayload(ctx, syntheticPayload, syntheticEnv.ParentBeaconBlockRoot)
	if err != nil {
		return common.Hash{}, fmt.Errorf("NewPayload failed: %w", err)
	}
	if status.Status != eth.ExecutionValid {
		validationErr := ""
		if status.ValidationError != nil {
			validationErr = *status.ValidationError
		}
		return common.Hash{}, fmt.Errorf("%w: status=%s validationError=%q",
			ErrRewindSyntheticPayloadRejected, status.Status, validationErr)
	}
	return syntheticPayload.BlockHash, nil
}

// buildSyntheticPayload clones envelope and perturbs ExtraData so the block hash
// differs from the original. ExtraData has no EVM opcode that reads it, so changing
// it will not change the state root — the engine will still accept the payload.
// The original envelope is not modified.
func buildSyntheticPayload(envelope *eth.ExecutionPayloadEnvelope) *eth.ExecutionPayloadEnvelope {
	newEnvelope := *envelope
	newPayload := *envelope.ExecutionPayload
	newEnvelope.ExecutionPayload = &newPayload

	extra := make([]byte, len(newPayload.ExtraData))
	copy(extra, newPayload.ExtraData)
	if len(extra) == 0 {
		extra = []byte{0x00}
	} else {
		extra[len(extra)-1] ^= 0xff
	}
	newPayload.ExtraData = extra
	syntheticHash, _ := newEnvelope.CheckBlockHash()
	newPayload.BlockHash = syntheticHash
	return &newEnvelope
}

// forkchoiceUpdateWithRetry sends an FCU and then verifies the engine's head/safe/finalized
// have converged. It retries up to maxFCUAttempts because reth can briefly report stale
// labels after an FCU (see reth#23205).
func forkchoiceUpdateWithRetry(ctx context.Context, lgr log.Logger, engineClient *sources.EngineAPIClient, open client.RPC, head, safe, finalized common.Hash) error {
	for attempt := 1; attempt <= maxFCUAttempts; attempt++ {
		if err := sendForkchoice(ctx, engineClient, head, safe, finalized); err != nil {
			return err
		}
		err := verifyRewindState(ctx, open, head, safe, finalized)
		if err == nil {
			return nil
		}
		if attempt == maxFCUAttempts {
			return fmt.Errorf("%w after %d attempts: %w", ErrRewindFCUHeadMismatch, maxFCUAttempts, err)
		}
		lgr.Warn("FCU state not yet converged, retrying", "attempt", attempt, "expectedHead", head, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(fcuRetryDelay):
		}
	}
	return nil // unreachable
}

func sendForkchoice(ctx context.Context, engineClient *sources.EngineAPIClient, head, safe, finalized common.Hash) error {
	res, err := engineClient.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{
		HeadBlockHash:      head,
		SafeBlockHash:      safe,
		FinalizedBlockHash: finalized,
	}, nil)
	if err != nil {
		return err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		validationErr := ""
		if res.PayloadStatus.ValidationError != nil {
			validationErr = *res.PayloadStatus.ValidationError
		}
		return fmt.Errorf("%w: status=%s validationError=%q head=%s safe=%s finalized=%s",
			ErrRewindFCURejected, res.PayloadStatus.Status, validationErr, head, safe, finalized)
	}
	return nil
}

// verifyRewindState confirms the engine's latest/safe/finalized hashes match the expected values.
// Hash equality is authoritative: if a hash matches, so does its block number.
func verifyRewindState(ctx context.Context, open client.RPC, head, safe, finalized common.Hash) error {
	latest, safeH, finalizedH, err := headSafeFinalized(ctx, open)
	if err != nil {
		return err
	}
	if latest == nil {
		return fmt.Errorf("unexpected latest: got nil, want %s", head)
	}
	if latest.Hash() != head {
		return fmt.Errorf("unexpected latest: got %s, want %s", latest.Hash(), head)
	}
	if safeH == nil {
		return fmt.Errorf("unexpected safe: got nil, want %s", safe)
	}
	if safeH.Hash() != safe {
		return fmt.Errorf("unexpected safe: got %s, want %s", safeH.Hash(), safe)
	}
	if finalizedH == nil {
		return fmt.Errorf("unexpected finalized: got nil, want %s", finalized)
	}
	if finalizedH.Hash() != finalized {
		return fmt.Errorf("unexpected finalized: got %s, want %s", finalizedH.Hash(), finalized)
	}
	return nil
}
