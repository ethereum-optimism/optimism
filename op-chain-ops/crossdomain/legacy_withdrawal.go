package crossdomain

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// LegacyWithdrawal represents a pre bedrock upgrade withdrawal.
type LegacyWithdrawal struct {
	// MessageSender is the caller of the message passer
	MessageSender common.Address `json:"who"`
	// XDomainTarget is the L1 target of the withdrawal message
	XDomainTarget common.Address `json:"target"`
	// XDomainSender is the L2 withdrawing account
	XDomainSender common.Address `json:"sender"`
	// XDomainData represents the calldata of the withdrawal message
	XDomainData hexutil.Bytes `json:"data"`
	// XDomainNonce represents the nonce of the withdrawal
	XDomainNonce *big.Int `json:"nonce"`
}

var _ WithdrawalMessage = (*LegacyWithdrawal)(nil)

// NewLegacyWithdrawal will construct a LegacyWithdrawal
func NewLegacyWithdrawal(msgSender, target, sender common.Address, data []byte, nonce *big.Int) *LegacyWithdrawal {
	return &LegacyWithdrawal{
		MessageSender: msgSender,
		XDomainTarget: target,
		XDomainSender: sender,
		XDomainData:   data,
		XDomainNonce:  nonce,
	}
}

// Encode will serialze the Withdrawal in the legacy format so that it
// is suitable for hashing. This assumes that the message is being withdrawn
// through the standard optimism cross domain messaging system by hashing in
// the L2CrossDomainMessenger address.
func (w *LegacyWithdrawal) Encode() ([]byte, error) {
	enc, err := EncodeCrossDomainMessageV0(w.XDomainTarget, w.XDomainSender, []byte(w.XDomainData), w.XDomainNonce)
	if err != nil {
		return nil, fmt.Errorf("cannot encode LegacyWithdrawal: %w", err)
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

	w.MessageSender = common.BytesToAddress(msgSender)
	w.XDomainTarget = target
	w.XDomainSender = sender
	w.XDomainData = msgData
	w.XDomainNonce = nonce
	return nil
}

// Hash will compute the legacy style hash that is computed in the
// OVM_L2ToL1MessagePasser.
func (w *LegacyWithdrawal) Hash() (common.Hash, error) {
	encoded, err := w.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot hash LegacyWithdrawal: %w", err)
	}
	hash := crypto.Keccak256(encoded)
	return common.BytesToHash(hash), nil
}

// StorageSlot will compute the storage slot that is set
// to true in the legacy L2ToL1MessagePasser.
func (w *LegacyWithdrawal) StorageSlot() (common.Hash, error) {
	hash, err := w.Hash()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot compute storage slot: %w", err)
	}
	preimage := make([]byte, 64)
	copy(preimage, hash.Bytes())

	slot := crypto.Keccak256(preimage)
	return common.BytesToHash(slot), nil
}

// Value returns the ETH value associated with the withdrawal. Since
// ETH was represented as an ERC20 token before the Bedrock upgrade,
// the sender and calldata must be observed and the value must be parsed
// out if "finalizeETHWithdrawal" is the method.
func (w *LegacyWithdrawal) Value() (*big.Int, error) {
	abi, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	value := new(big.Int)

	// Parse the 4byte selector
	method, err := abi.MethodById(w.XDomainData)
	// If it is an unknown selector, there is no value
	if err != nil {
		return value, nil
	}

	isFromL2StandardBridge := w.XDomainSender == predeploys.L2StandardBridgeAddr
	if isFromL2StandardBridge && method.Name == "finalizeETHWithdrawal" {
		data, err := method.Inputs.Unpack(w.XDomainData[4:])
		if err != nil {
			return nil, err
		}
		// bounds check
		if len(data) < 3 {
			return nil, errors.New("not enough data")
		}
		var ok bool
		value, ok = data[2].(*big.Int)
		if !ok {
			return nil, errors.New("not big.Int")
		}
	}

	return value, nil
}

// CrossDomainMessage turns the LegacyWithdrawal into
// a CrossDomainMessage. LegacyWithdrawals do not have
// the concept of value or gaslimit, so set them to 0.
func (w *LegacyWithdrawal) CrossDomainMessage() *CrossDomainMessage {
	return &CrossDomainMessage{
		Nonce:    w.XDomainNonce,
		Sender:   w.XDomainSender,
		Target:   w.XDomainTarget,
		Value:    new(big.Int),
		GasLimit: new(big.Int),
		Data:     []byte(w.XDomainData),
	}
}
