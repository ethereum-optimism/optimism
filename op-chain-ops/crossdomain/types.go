package crossdomain

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// DangerousUnfilteredWithdrawals is a list of raw withdrawal witness
// data. It has not been filtered for messages from sources other than
// the
type DangerousUnfilteredWithdrawals []*LegacyWithdrawal

// SafeFilteredWithdrawals is a list of withdrawals that have been filtered to only include
// withdrawals that were from the L2XDM.
type SafeFilteredWithdrawals []*LegacyWithdrawal

var (
	// Standard ABI types
	Uint256Type, _ = abi.NewType("uint256", "", nil)
	BytesType, _   = abi.NewType("bytes", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)
	Bytes32Type, _ = abi.NewType("bytes32", "", nil)
)

// WithdrawalMessage represents a Withdrawal. The Withdrawal
// and LegacyWithdrawal types must implement this interface.
type WithdrawalMessage interface {
	Encode() ([]byte, error)
	Decode([]byte) error
	Hash() (common.Hash, error)
	StorageSlot() (common.Hash, error)
}
