package crossdomain

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// CrossDomainMessage represents a cross domain message
// used by the CrossDomainMessenger. The version is encoded
// in the nonce. Version 0 messages do not have a value,
// version 1 messages have a value and the most significant
// byte of the nonce is a 1
type CrossDomainMessage struct {
	Nonce    *big.Int
	Sender   *common.Address
	Target   *common.Address
	Value    *big.Int
	GasLimit *big.Int
	Data     []byte
}

// NewCrossDomainMessage creates a CrossDomainMessage.
func NewCrossDomainMessage(
	nonce *big.Int,
	sender, target *common.Address,
	value, gasLimit *big.Int,
	data []byte,
) *CrossDomainMessage {
	return &CrossDomainMessage{
		Nonce:    nonce,
		Sender:   sender,
		Target:   target,
		Value:    value,
		GasLimit: gasLimit,
		Data:     data,
	}
}

// Version will return the version of the CrossDomainMessage.
// It does this by looking at the first byte of the nonce.
func (c *CrossDomainMessage) Version() uint64 {
	_, version := DecodeVersionedNonce(c.Nonce)
	return version.Uint64()
}

// Encode will encode a cross domain message based on the version.
func (c *CrossDomainMessage) Encode() ([]byte, error) {
	version := c.Version()
	switch version {
	case 0:
		return EncodeCrossDomainMessageV0(c.Target, c.Sender, c.Data, c.Nonce)
	case 1:
		return EncodeCrossDomainMessageV1(c.Nonce, c.Sender, c.Target, c.Value, c.GasLimit, c.Data)
	default:
		return nil, fmt.Errorf("unknown version %d", version)
	}
}

// Hash will compute the hash of the CrossDomainMessage
func (c *CrossDomainMessage) Hash() (common.Hash, error) {
	version := c.Version()
	switch version {
	case 0:
		return HashCrossDomainMessageV0(c.Target, c.Sender, c.Data, c.Nonce)
	case 1:
		return HashCrossDomainMessageV1(c.Nonce, c.Sender, c.Target, c.Value, c.GasLimit, c.Data)
	default:
		return common.Hash{}, fmt.Errorf("unknown version %d", version)
	}
}

// ToWithdrawal will turn a CrossDomainMessage into a Withdrawal.
// This only works for version 0 CrossDomainMessages as not all of
// the data is present for version 1 CrossDomainMessages to be turned
// into Withdrawals.
func (c *CrossDomainMessage) ToWithdrawal() (WithdrawalMessage, error) {
	version := c.Version()
	switch version {
	case 0:
		if c.Value != nil && c.Value.Cmp(common.Big0) != 0 {
			return nil, errors.New("version 0 messages must have 0 value")
		}
		w := NewLegacyWithdrawal(c.Target, c.Sender, c.Data, c.Nonce)
		return w, nil
	case 1:
		return nil, errors.New("version 1 messages cannot be turned into withdrawals")
	default:
		return nil, fmt.Errorf("unknown version %d", version)
	}
}
