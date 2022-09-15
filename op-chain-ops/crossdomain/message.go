package crossdomain

import (
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
		return nil, fmt.Errorf("unknown nonce version %d", version)
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
		return common.Hash{}, fmt.Errorf("unknown nonce version %d", version)
	}
}
