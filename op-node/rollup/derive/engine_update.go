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

// InsertHeadBlock creates, executes, and inserts the specified block as the head block.
// It first uses the given FC to start the block creation process and then after the payload is executed,
// sets the FC to the same safe and finalized hashes, but updates the head hash to the new block.
// If updateSafe is true, the head block is considered to be the safe head as well as the head.
// It returns the payload, an RPC error (if the payload might still be valid), and a payload error (if the payload was not valid)
func InsertHeadBlock(ctx context.Context, log log.Logger, eng Engine, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, updateSafe bool) (out *eth.ExecutionPayload, rpcErr error, payloadErr error) {
	fcRes, err := eng.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block via forkchoice: %w", err), nil
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus), nil
	}
	id := fcRes.PayloadID
	if id == nil {
		return nil, errors.New("nil id in forkchoice result when expecting a valid ID"), nil
	}
	payload, err := eng.GetPayload(ctx, *id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution payload: %w", err), nil
	}
	if err := sanityCheckPayload(payload); err != nil {
		return nil, nil, err
	}

	status, err := eng.NewPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to insert execution payload: %w", err), nil
	}
	if status.Status != eth.ExecutionValid {
		return nil, eth.NewPayloadErr(payload, status), nil
	}

	fc.HeadBlockHash = payload.BlockHash
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	fcRes, err = eng.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err), nil
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, eth.ForkchoiceUpdateErr(fcRes.PayloadStatus), nil
	}
	log.Info("inserted block", "hash", payload.BlockHash, "number", uint64(payload.BlockNumber),
		"state_root", payload.StateRoot, "timestamp", uint64(payload.Timestamp), "parent", payload.ParentHash,
		"prev_randao", payload.PrevRandao, "fee_recipient", payload.FeeRecipient,
		"txs", len(payload.Transactions), "update_safe", updateSafe)
	return payload, nil, nil
}
