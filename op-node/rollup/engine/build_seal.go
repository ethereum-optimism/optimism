package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PayloadSealInvalidEvent identifies a permanent in-consensus problem with the payload sealing.
type PayloadSealInvalidEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealInvalidEvent) String() string {
	return "payload-seal-invalid"
}

// PayloadSealExpiredErrorEvent identifies a form of failed payload-sealing that is not coupled
// to the attributes themselves, but rather the build-job process.
// The user should re-attempt by starting a new build process. The payload-sealing job should not be re-attempted,
// as it most likely expired, timed out, or referenced an otherwise invalidated block-building job identifier.
type PayloadSealExpiredErrorEvent struct {
	Info eth.PayloadInfo
	Err  error

	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (ev PayloadSealExpiredErrorEvent) String() string {
	return "payload-seal-expired-error"
}

type BuildSealEvent struct {
	Info         eth.PayloadInfo
	BuildStarted time.Time
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildSealEvent) String() string {
	return "build-seal"
}

func (e *EngineController) onBuildSeal(ctx context.Context, ev BuildSealEvent) {
	result, err := e.sealBuild(ctx, ev.Info, ev.BuildStarted, ev.Concluding, ev.DerivedFrom)
	if err != nil {
		return
	}
	e.emitter.Emit(ctx, BuildSealedEvent{
		Concluding:   ev.Concluding,
		DerivedFrom:  ev.DerivedFrom,
		BuildStarted: ev.BuildStarted,
		Info:         ev.Info,
		Envelope:     result.Envelope,
		Ref:          result.Ref,
	})
}

// sealBuild contains the core logic for sealing a block-building job.
// It does NOT acquire e.mu (caller is responsible).
// Emits error events (PayloadSealExpiredErrorEvent, PayloadSealInvalidEvent) but NOT BuildSealedEvent (caller's job).
func (e *EngineController) sealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time, concluding bool, derivedFrom eth.L1BlockRef) (*SealResult, error) {
	rpcCtx, cancel := context.WithTimeout(e.ctx, buildSealTimeout)
	defer cancel()

	sealingStart := time.Now()
	envelope, err := e.engine.GetPayload(rpcCtx, info)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			e.log.Warn("Cannot seal block, payload ID is unknown",
				"payloadID", info.ID, "payload_time", info.Timestamp,
				"started_time", buildStarted)
		}
		e.emitter.Emit(ctx, PayloadSealExpiredErrorEvent{
			Info:        info,
			Err:         fmt.Errorf("failed to seal execution payload (ID: %s): %w", info.ID, err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, fmt.Errorf("failed to seal execution payload (ID: %s): %w", info.ID, err)
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		e.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info: info,
			Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
				info.ID, envelope.ExecutionPayload.BlockHash, err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
			info.ID, envelope.ExecutionPayload.BlockHash, err)
	}

	ref, err := derive.PayloadToBlockRef(e.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		e.emitter.Emit(ctx, PayloadSealInvalidEvent{
			Info:        info,
			Err:         fmt.Errorf("failed to decode L2 block ref from payload: %w", err),
			Concluding:  concluding,
			DerivedFrom: derivedFrom,
		})
		return nil, fmt.Errorf("failed to decode L2 block ref from payload: %w", err)
	}

	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(buildStarted)
	e.metrics.RecordSequencerSealingTime(sealTime)
	e.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(e.rollupCfg.BlockTime)*time.Second)

	txnCount := len(envelope.ExecutionPayload.Transactions)
	depositCount, _ := lastDeposit(envelope.ExecutionPayload.Transactions)
	e.metrics.CountSequencedTxsInBlock(txnCount, depositCount)

	e.log.Debug("Built new L2 block", "l2_unsafe", ref, "l1_origin", ref.L1Origin,
		"txs", txnCount, "deposits", depositCount, "time", ref.Time, "seal_time", sealTime, "build_time", buildTime)

	return &SealResult{
		Envelope: envelope,
		Ref:      ref,
	}, nil
}

// SealBuild retrieves a built payload via GetPayload directly, bypassing the event system.
// Acquires e.mu. Returns the sealed envelope and block ref.
// On error, emits PayloadSealExpiredErrorEvent or PayloadSealInvalidEvent before returning.
// Does NOT emit BuildSealedEvent.
func (e *EngineController) SealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*SealResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sealBuild(ctx, info, buildStarted, false, eth.L1BlockRef{})
}

// isDepositTx checks an opaqueTx to determine if it is a Deposit Transaction
// It has to return an error in the case the transaction is empty
func isDepositTx(opaqueTx eth.Data) (bool, error) {
	if len(opaqueTx) == 0 {
		return false, errors.New("empty transaction")
	}
	return opaqueTx[0] == types.DepositTxType, nil
}

// lastDeposit finds the index of last deposit at the start of the transactions.
// It walks the transactions from the start until it finds a non-deposit tx.
// An error is returned if any looked at transaction cannot be decoded
func lastDeposit(txns []eth.Data) (int, error) {
	var lastDeposit int
	for i, tx := range txns {
		deposit, err := isDepositTx(tx)
		if err != nil {
			return 0, fmt.Errorf("invalid transaction at idx %d", i)
		}
		if deposit {
			lastDeposit = i
		} else {
			break
		}
	}
	return lastDeposit, nil
}

func sanityCheckPayload(payload *eth.ExecutionPayload) error {
	// Sanity check payload before inserting it
	if len(payload.Transactions) == 0 {
		return errors.New("no transactions in returned payload")
	}
	if payload.Transactions[0][0] != types.DepositTxType {
		return fmt.Errorf("first transaction was not deposit tx. Got %v", payload.Transactions[0][0])
	}
	// Ensure that the deposits are first
	lastDeposit, err := lastDeposit(payload.Transactions)
	if err != nil {
		return fmt.Errorf("failed to find last deposit: %w", err)
	}
	// Ensure no deposits after last deposit
	for i := lastDeposit + 1; i < len(payload.Transactions); i++ {
		tx := payload.Transactions[i]
		deposit, err := isDepositTx(tx)
		if err != nil {
			return fmt.Errorf("failed to decode transaction idx %d: %w", i, err)
		}
		if deposit {
			return fmt.Errorf("deposit tx (%d) after other tx in l2 block with prev deposit at idx %d", i, lastDeposit)
		}
	}
	return nil
}
