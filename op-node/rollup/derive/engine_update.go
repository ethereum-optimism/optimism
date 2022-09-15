package derive

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

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

type BlockInsertionErrType uint

const (
	BlockInsertOK BlockInsertionErrType = iota
	BlockInsertTemporaryErr
	BlockInsertPrestateErr
	BlockInsertPayloadErr
)

// InsertHeadBlock creates, executes, and inserts the specified block as the head block.
// It first uses the given FC to start the block creation process and then after the payload is executed,
// sets the FC to the same safe and finalized hashes, but updates the head hash to the new block.
// If updateSafe is true, the head block is considered to be the safe head as well as the head.
// It returns the payload, an RPC error (if the payload might still be valid), and a payload error (if the payload was not valid)
func InsertHeadBlock(ctx context.Context, log log.Logger, eng Engine, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, updateSafe bool) (out *eth.ExecutionPayload, errTyp BlockInsertionErrType, err error) {
	fcRes, err := eng.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				return nil, BlockInsertPrestateErr, fmt.Errorf("pre-block-creation forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap())
			case eth.InvalidPayloadAttributes:
				return nil, BlockInsertPayloadErr, fmt.Errorf("payload attributes are not valid, cannot build block: %w", inputErr.Unwrap())
			default:
				return nil, BlockInsertPrestateErr, fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err)
			}
		} else {
			return nil, BlockInsertTemporaryErr, fmt.Errorf("failed to create new block via forkchoice: %w", err)
		}
	}

	switch fcRes.PayloadStatus.Status {
	// TODO(proto): snap sync - specify explicit different error type if node is syncing
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		return nil, BlockInsertPayloadErr, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
	case eth.ExecutionValid:
		break
	default:
		return nil, BlockInsertTemporaryErr, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
	}
	id := fcRes.PayloadID
	if id == nil {
		return nil, BlockInsertTemporaryErr, errors.New("nil id in forkchoice result when expecting a valid ID")
	}
	payload, err := eng.GetPayload(ctx, *id)
	if err != nil {
		// even if it is an input-error (unknown payload ID), it is temporary, since we will re-attempt the full payload building, not just the retrieval of the payload.
		return nil, BlockInsertTemporaryErr, fmt.Errorf("failed to get execution payload: %w", err)
	}
	if err := sanityCheckPayload(payload); err != nil {
		return nil, BlockInsertPayloadErr, err
	}

	status, err := eng.NewPayload(ctx, payload)
	if err != nil {
		return nil, BlockInsertTemporaryErr, fmt.Errorf("failed to insert execution payload: %w", err)
	}
	if status.Status == eth.ExecutionInvalid || status.Status == eth.ExecutionInvalidBlockHash {
		return nil, BlockInsertPayloadErr, eth.NewPayloadErr(payload, status)
	}
	if status.Status != eth.ExecutionValid {
		return nil, BlockInsertTemporaryErr, eth.NewPayloadErr(payload, status)
	}

	fc.HeadBlockHash = payload.BlockHash
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	fcRes, err = eng.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				// if we succeed to update the forkchoice pre-payload, but fail post-payload, then it is a payload error
				return nil, BlockInsertPayloadErr, fmt.Errorf("post-block-creation forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap())
			default:
				return nil, BlockInsertPrestateErr, fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err)
			}
		} else {
			return nil, BlockInsertTemporaryErr, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err)
		}
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, BlockInsertPayloadErr, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)
	}
	log.Info("inserted block", "hash", payload.BlockHash, "number", uint64(payload.BlockNumber),
		"state_root", payload.StateRoot, "timestamp", uint64(payload.Timestamp), "parent", payload.ParentHash,
		"prev_randao", payload.PrevRandao, "fee_recipient", payload.FeeRecipient,
		"txs", len(payload.Transactions), "update_safe", updateSafe)
	return payload, BlockInsertOK, nil
}
