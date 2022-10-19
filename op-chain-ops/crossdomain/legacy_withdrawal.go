package crossdomain

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// LegacyWithdrawal represents a pre bedrock upgrade withdrawal.
type LegacyWithdrawal struct {
	Target *common.Address `json:"target"`
	Sender *common.Address `json:"sender"`
	Data   []byte          `json:"data"`
	Nonce  *big.Int        `json:"nonce"`
}

var _ WithdrawalMessage = (*LegacyWithdrawal)(nil)

// NewLegacyWithdrawal will construct a LegacyWithdrawal
func NewLegacyWithdrawal(target, sender *common.Address, data []byte, nonce *big.Int) *LegacyWithdrawal {
	return &LegacyWithdrawal{
		Target: target,
		Sender: sender,
		Data:   data,
		Nonce:  nonce,
	}
}

// Encode will serialze the Withdrawal in the legacy format so that it
// is suitable for hashing. This assumes that the message is being withdrawn
// through the standard optimism cross domain messaging system by hashing in
// the L2CrossDomainMessenger address.
func (w *LegacyWithdrawal) Encode() ([]byte, error) {
	enc, err := EncodeCrossDomainMessageV0(w.Target, w.Sender, w.Data, w.Nonce)
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(enc)+len(predeploys.L2CrossDomainMessengerAddr.Bytes()))
	copy(out, enc)
	copy(out[len(enc):], predeploys.L2CrossDomainMessengerAddr.Bytes())
	return out, nil
}

// Decode will decode a serialized LegacyWithdrawal
func (w *LegacyWithdrawal) Decode(data []byte) error {
	if len(data) < len(predeploys.L2CrossDomainMessengerAddr)+4 {
		return fmt.Errorf("withdrawal data too short: %d", len(data))
	}

	selector := crypto.Keccak256([]byte("relayMessage(address,address,bytes,uint256)"))[0:4]
	if !bytes.Equal(data[0:4], selector) {
		return fmt.Errorf("invalid selector: 0x%x", data[0:4])
	}

	msgSender := data[len(data)-len(predeploys.L2CrossDomainMessengerAddr):]
	if !bytes.Equal(msgSender, predeploys.L2CrossDomainMessengerAddr.Bytes()) {
		return errors.New("invalid msg.sender")
	}

	raw := data[4 : len(data)-len(predeploys.L2CrossDomainMessengerAddr)]

	args := abi.Arguments{
		{Name: "target", Type: AddressType},
		{Name: "sender", Type: AddressType},
		{Name: "data", Type: BytesType},
		{Name: "nonce", Type: Uint256Type},
	}

	decoded, err := args.Unpack(raw)
	if err != nil {
		return err
	}

	target, ok := decoded[0].(common.Address)
	if !ok {
		return errors.New("cannot abi decode target")
	}
	sender, ok := decoded[1].(common.Address)
	if !ok {
		return errors.New("cannot abi decode sender")
	}
	msgData, ok := decoded[2].([]byte)
	if !ok {
		return errors.New("cannot abi decode data")
	}
	nonce, ok := decoded[3].(*big.Int)
	if !ok {
		return errors.New("cannot abi decode nonce")
	}

	w.Target = &target
	w.Sender = &sender
	w.Data = msgData
	w.Nonce = nonce
	return nil
}

// Hash will compute the legacy style hash that is computed in the
// OVM_L2ToL1MessagePasser.
func (w *LegacyWithdrawal) Hash() (common.Hash, error) {
	encoded, err := w.Encode()
	if err != nil {
		return common.Hash{}, nil
	}
	hash := crypto.Keccak256(encoded)
	return common.BytesToHash(hash), nil
}

// StorageSlot will compute the storage slot that is set
// to true in the legacy L2ToL1MessagePasser.
func (w *LegacyWithdrawal) StorageSlot() (common.Hash, error) {
	hash, err := w.Hash()
	if err != nil {
		return common.Hash{}, err
	}
	preimage := make([]byte, 64)
	copy(preimage, hash.Bytes())

	slot := crypto.Keccak256(preimage)
	return common.BytesToHash(slot), nil
}
