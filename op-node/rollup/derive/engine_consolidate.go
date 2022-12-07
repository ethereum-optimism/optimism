package derive

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// AttributesMatchBlock checks if the L2 attributes pre-inputs match the output
// nil if it is a match. If err is not nil, the error contains the reason for the mismatch
func AttributesMatchBlock(attrs *eth.PayloadAttributes, parentHash common.Hash, block *eth.ExecutionPayload) error {
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
