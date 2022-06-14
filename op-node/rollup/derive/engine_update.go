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

// InsertHeadBlock creates, executes, and inserts the specified block as the head block.
// It first uses the given FC to start the block creation process and then after the payload is executed,
// sets the FC to the same safe and finalized hashes, but updates the head hash to the new block.
// If updateSafe is true, the head block is considered to be the safe head as well as the head.
// It returns the payload, the count of deposits, and an error.
func InsertHeadBlock(ctx context.Context, log log.Logger, eng Engine, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, updateSafe bool) (*eth.ExecutionPayload, error) {
	fcRes, err := eng.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block via forkchoice: %w", err)
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("engine not ready, forkchoice pre-state is not valid: %s", fcRes.PayloadStatus.Status)
	}
	id := fcRes.PayloadID
	if id == nil {
		return nil, errors.New("nil id in forkchoice result when expecting a valid ID")
	}
	payload, err := eng.GetPayload(ctx, *id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution payload: %w", err)
	}
	// Sanity check payload before inserting it
	if len(payload.Transactions) == 0 {
		return nil, errors.New("no transactions in returned payload")
	}
	if payload.Transactions[0][0] != types.DepositTxType {
		return nil, fmt.Errorf("first transaction was not deposit tx. Got %v", payload.Transactions[0][0])
	}
	// Ensure that the deposits are first
	lastDeposit, err := lastDeposit(payload.Transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to find last deposit: %w", err)
	}
	// Ensure no deposits after last deposit
	for i := lastDeposit + 1; i < len(payload.Transactions); i++ {
		tx := payload.Transactions[i]
		deposit, err := isDepositTx(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction idx %d: %w", i, err)
		}
		if deposit {
			log.Error("Produced an invalid block where the deposit txns are not all at the start of the block", "tx_idx", i, "lastDeposit", lastDeposit)
			return nil, fmt.Errorf("deposit tx (%d) after other tx in l2 block with prev deposit at idx %d", i, lastDeposit)
		}
	}

	err = eng.NewPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to insert execution payload: %w", err)
	}
	fc.HeadBlockHash = payload.BlockHash
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	log.Debug("Inserted L2 head block", "number", uint64(payload.BlockNumber), "hash", payload.BlockHash, "update_safe", updateSafe)
	fcRes, err = eng.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err)
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("failed to persist forkchoice change: %s", fcRes.PayloadStatus.Status)
	}
	return payload, nil
}
