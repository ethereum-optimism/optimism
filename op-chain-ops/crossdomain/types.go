package crossdomain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

// InvalidMessage represents a message to the L1 message passer that
// cannot be decoded as a withdrawal. They are defined as a separate
// type in order to completely disambiguate them from any other
// message.
type InvalidMessage SentMessage

func (msg *InvalidMessage) Encode() ([]byte, error) {
	out := make([]byte, len(msg.Msg)+20)
	copy(out, msg.Msg)
	copy(out[len(msg.Msg):], msg.Who.Bytes())
	return out, nil
}

func (msg *InvalidMessage) Hash() (common.Hash, error) {
	bytes, err := msg.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot hash: %w", err)
	}
	return crypto.Keccak256Hash(bytes), nil
}

func (msg *InvalidMessage) StorageSlot() (common.Hash, error) {
	hash, err := msg.Hash()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot compute storage slot: %w", err)
	}
	preimage := make([]byte, 64)
	copy(preimage, hash.Bytes())

	return crypto.Keccak256Hash(preimage), nil
}
