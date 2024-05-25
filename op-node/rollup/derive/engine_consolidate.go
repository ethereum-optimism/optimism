package derive

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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
				logL1InfoTxns(l, uint64(block.BlockNumber), uint64(block.Timestamp), otx, block.Transactions[i])
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
	if withdrawalErr := checkWithdrawalsMatch(attrs.Withdrawals, block.Withdrawals); withdrawalErr != nil {
		return withdrawalErr
	}
	return nil
}

func checkWithdrawalsMatch(attrWithdrawals *types.Withdrawals, blockWithdrawals *types.Withdrawals) error {
	if attrWithdrawals == nil && blockWithdrawals == nil {
		return nil
	}

	if attrWithdrawals == nil && blockWithdrawals != nil {
		return fmt.Errorf("expected withdrawals in block to be nil, actual %v", *blockWithdrawals)
	}

	if attrWithdrawals != nil && blockWithdrawals == nil {
		return fmt.Errorf("expected withdrawals in block to be non-nil %v, actual nil", *attrWithdrawals)
	}

	if len(*attrWithdrawals) != len(*blockWithdrawals) {
		return fmt.Errorf("expected withdrawals in block to be %d, actual %d", len(*attrWithdrawals), len(*blockWithdrawals))
	}

	for idx, expected := range *attrWithdrawals {
		actual := (*blockWithdrawals)[idx]

		if *expected != *actual {
			return fmt.Errorf("expected withdrawal %d to be %v, actual %v", idx, expected, actual)
		}
	}

	return nil
}

// logL1InfoTxns reports the values from the L1 info tx when they differ to aid
// debugging. This check is the one that has been most frequently triggered.
func logL1InfoTxns(l log.Logger, l2Number, l2Timestamp uint64, safeTx, unsafeTx hexutil.Bytes) {
	// First decode into *types.Transaction to get the tx data.
	var safeTxValue, unsafeTxValue types.Transaction
	errSafe := (&safeTxValue).UnmarshalBinary(safeTx)
	errUnsafe := (&unsafeTxValue).UnmarshalBinary(unsafeTx)
	if errSafe != nil || errUnsafe != nil {
		l.Error("failed to umarshal tx", "errSafe", errSafe, "errUnsafe", errUnsafe)
		return
	}

	// Then decode the ABI encoded parameters
	var safeInfo, unsafeInfo L1BlockInfo
	errSafe = (&safeInfo).UnmarshalBinary(safeTxValue.Data())
	errUnsafe = (&unsafeInfo).UnmarshalBinary(unsafeTxValue.Data())
	if errSafe != nil || errUnsafe != nil {
		l.Error("failed to umarshal l1 info", "errSafe", errSafe, "errUnsafe", errUnsafe)
		return
	}

	l.Error("L1 Info transaction differs", "number", l2Number, "time", l2Timestamp,
		"safe_l1_number", safeInfo.Number, "safe_l1_hash", safeInfo.BlockHash,
		"safe_l1_time", safeInfo.Time, "safe_seq_num", safeInfo.SequenceNumber,
		"safe_l1_basefee", safeInfo.BaseFee, "safe_batcher_add", safeInfo.BlockHash,
		"safe_gpo_scalar", safeInfo.L1FeeScalar, "safe_gpo_overhead", safeInfo.L1FeeOverhead,
		"unsafe_l1_number", unsafeInfo.Number, "unsafe_l1_hash", unsafeInfo.BlockHash,
		"unsafe_l1_time", unsafeInfo.Time, "unsafe_seq_num", unsafeInfo.SequenceNumber,
		"unsafe_l1_basefee", unsafeInfo.BaseFee, "unsafe_batcher_add", unsafeInfo.BlockHash,
		"unsafe_gpo_scalar", unsafeInfo.L1FeeScalar, "unsafe_gpo_overhead", unsafeInfo.L1FeeOverhead)
}
