package wire

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// MaxTxGas is the projection transaction gas ceiling (EIP-7825).
const MaxTxGas = 16777216

const MaxMessageBytes = 1024 * 1024

var ValidateMessageSelector = selector("validateMessage((address,uint256,uint256,uint256,uint256),bytes32)")

// DecodeClaim parses bounded canonical calldata without interpreting its proof.
func DecodeClaim(data []byte) (*codec.RangeClaim, error) {
	if len(data) < 4 || !bytes.Equal(data[:4], PostClaimSelector[:]) {
		return nil, fmt.Errorf("invalid claim selector")
	}
	return codec.DecodeMode(data[4:], codec.ModeProven)
}

func canonicalArgs(data []byte, selector [4]byte, args abi.Arguments) ([]any, error) {
	if len(data) < 4 || len(data) > MaxMessageBytes+1024 || !bytes.Equal(data[:4], selector[:]) {
		return nil, fmt.Errorf("invalid replay selector or length")
	}
	values, err := args.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	encoded, err := args.Pack(values...)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, data[4:]) {
		return nil, fmt.Errorf("noncanonical replay calldata")
	}
	return values, nil
}

func DecodeReplaySentMessage(data []byte) (*SentMessage, error) {
	v, err := canonicalArgs(data, ReplaySentMessageSelector, ReplaySentMessageArgs)
	if err != nil {
		return nil, err
	}
	m := &SentMessage{Destination: v[0].(*big.Int), Nonce: v[1].(*big.Int), Sender: v[2].(common.Address), Target: v[3].(common.Address), Message: v[4].([]byte)}
	if len(m.Message) > MaxMessageBytes {
		return nil, fmt.Errorf("message too large")
	}
	return m, nil
}

func CheckReplayEvent(data []byte) error {
	v, err := canonicalArgs(data, ReplayEventSelector, ReplayEventArgs)
	if err != nil {
		return err
	}
	if len(v[0].([][32]byte)) > 4 || len(v[1].([]byte)) > MaxMessageBytes {
		return fmt.Errorf("event exceeds bounds")
	}
	return nil
}

// DecodeValidateMessage uses the ABI's six static words. Narrowing is checked before conversion.
func DecodeValidateMessage(data []byte) (*messages.Message, error) {
	if len(data) != 4+6*32 || !bytes.Equal(data[:4], ValidateMessageSelector[:]) {
		return nil, fmt.Errorf("invalid validateMessage calldata")
	}
	w := data[4:]
	origin := common.BytesToAddress(w[:32])
	if common.BytesToHash(w[:32]) != common.BytesToHash(origin[:]) {
		return nil, fmt.Errorf("noncanonical origin")
	}
	number, index, timestamp := new(big.Int).SetBytes(w[32:64]), new(big.Int).SetBytes(w[64:96]), new(big.Int).SetBytes(w[96:128])
	if !number.IsUint64() || !index.IsUint64() || index.BitLen() > 32 || !timestamp.IsUint64() {
		return nil, fmt.Errorf("message identifier overflows")
	}
	return &messages.Message{Identifier: messages.Identifier{Origin: origin, BlockNumber: bigs.Uint64Strict(number), LogIndex: uint32(bigs.Uint64Strict(index)), Timestamp: bigs.Uint64Strict(timestamp), ChainID: eth.ChainIDFromBytes32([32]byte(w[128:160]))}, PayloadHash: common.BytesToHash(w[160:192])}, nil
}
