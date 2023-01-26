package derive

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// AttributesMatchBlock checks if the L2 attributes pre-inputs match the output
// nil if it is a match. If err is not nil, the error contains the reason for the mismatch
func AttributesMatchBlock(attrs *eth.PayloadAttributes, parentHash common.Hash, block *eth.ExecutionPayload, l log.Logger) error {
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
		return fmt.Errorf("transaction count does not match. expected: %d. got: %d", len(attrs.Transactions), len(block.Transactions))
	}
	for i, otx := range attrs.Transactions {
		if expect := block.Transactions[i]; !bytes.Equal(otx, expect) {
			if i == 0 {
				var safeTx, unsafeTx types.Transaction
				var safeInfo, unsafeInfo L1BlockInfo
				errSafe := (&safeTx).UnmarshalBinary(otx)
				errUnsafe := (&unsafeTx).UnmarshalBinary(otx)
				if errSafe == nil && errUnsafe == nil {
					errSafe = (&safeInfo).UnmarshalBinary(safeTx.Data())
					errUnsafe = (&unsafeInfo).UnmarshalBinary(unsafeTx.Data())
					if errSafe == nil && errUnsafe == nil {
						l.Warn("L1 Info transaction differs", "number", uint64(block.BlockNumber), "time", uint64(block.Timestamp),
							"safe_l1_number", safeInfo.Number, "safe_l1_hash", safeInfo.BlockHash,
							"safe_l1_time", safeInfo.Time, "safe_seq_num", safeInfo.SequenceNumber,
							"unsafe_l1_number", unsafeInfo.Number, "unsafe_l1_hash", unsafeInfo.BlockHash,
							"unsafe_l1_time", unsafeInfo.Time, "unsafe_seq_num", unsafeInfo.SequenceNumber)
					} else {
						l.Warn("failed to umarshal l1 info", "errSafe", errSafe, "errUnsafe", errUnsafe)
					}
				} else {
					l.Warn("failed to umarshal tx", "errSafe", errSafe, "errUnsafe", errUnsafe)
				}
			}
			return fmt.Errorf("transaction %d does not match. expected: %v. got: %v", i, expect, otx)
		}
	}
	if attrs.GasLimit == nil {
		return fmt.Errorf("expected gaslimit in attributes to not be nil, expected %d", block.GasLimit)
	}
	if *attrs.GasLimit != block.GasLimit {
		return fmt.Errorf("gas limit does not match. expected %d. got: %d", *attrs.GasLimit, block.GasLimit)
	}
	return nil
}
