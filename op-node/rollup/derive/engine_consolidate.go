package derive

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

// attributesMatchBlock checks if the L2 attributes pre-inputs match the output
// nil if it is a match. If err is not nil, the error contains the reason for the mismatch
func attributesMatchBlock(attrs *eth.PayloadAttributes, parentHash common.Hash, block *eth.ExecutionPayload) error {
	if parentHash != block.ParentHash {
		return fmt.Errorf("parent hash field does not match. expected: %v. got: %v", parentHash, block.ParentHash)
	}
	if attrs.Timestamp != block.Timestamp {
		return fmt.Errorf("timestamp field does not match. expected: %v. got: %v", uint64(attrs.Timestamp), block.Timestamp)
	}
	if attrs.PrevRandao != block.PrevRandao {
		return fmt.Errorf("random field does not match. expected: %v. got: %v", attrs.PrevRandao, block.PrevRandao)
	}
	if len(attrs.Transactions) != len(block.Transactions) {
		return fmt.Errorf("transaction count does not match. expected: %v. got: %v", len(attrs.Transactions), block.Transactions)
	}
	for i, otx := range attrs.Transactions {
		if expect := block.Transactions[i]; !bytes.Equal(otx, expect) {
			return fmt.Errorf("transaction %d does not match. expected: %v. got: %v", i, expect, otx)
		}
	}
	return nil
}

// VerifySafeBlock reconciles the supplied payload attributes against the actual L2 block.
// If they do not match, it inserts the new block and sets the head and safe head to the new block in the FC.
func VerifySafeBlock(ctx context.Context, log log.Logger, eng Engine, genesis *rollup.Genesis, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, parent eth.BlockID) (*eth.ExecutionPayload, bool, error) {
	payload, err := eng.PayloadByNumber(ctx, new(big.Int).SetUint64(parent.Number+1))
	if err != nil {
		return nil, false, fmt.Errorf("failed to get L2 block: %w", err)
	}
	ref, err := PayloadToBlockRef(payload, genesis)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse block ref: %w", err)
	}
	log.Debug("verifySafeBlock", "parentl2", parent, "payload", payload.ID(), "payloadOrigin", ref.L1Origin, "payloadSeq", ref.SequenceNumber)
	err = attributesMatchBlock(attrs, parent.Hash, payload)
	if err != nil {
		// Have reorg
		log.Warn("Detected L2 reorg when verifying L2 safe head", "parent", parent, "prev_block", payload.BlockHash, "mismatch", err)
		fc.HeadBlockHash = parent.Hash
		fc.SafeBlockHash = parent.Hash
		payload, err := InsertHeadBlock(ctx, log, eng, fc, attrs, true)
		return payload, true, err
	}
	// If the attributes match, just bump the safe head
	log.Debug("Verified L2 block", "number", payload.BlockNumber, "hash", payload.BlockHash)
	fc.SafeBlockHash = payload.BlockHash
	_, err = eng.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to execute ForkchoiceUpdated: %w", err)
	}
	return payload, false, nil
}
