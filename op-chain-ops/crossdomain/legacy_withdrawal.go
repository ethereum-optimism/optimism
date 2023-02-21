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
	// Target is the L1 target of the withdrawal message
	Target *common.Address `json:"target"`
	// Sender is the L2 withdrawing account
	Sender *common.Address `json:"sender"`
	// Data represents the calldata of the withdrawal message
	Data hexutil.Bytes `json:"data"`
	// Nonce represents the nonce of the withdrawal
	Nonce *big.Int `json:"nonce"`
	// MessageSender represents the caller of the LegacyMessagePasser
	MessageSender common.Address `json:"messageSender"`
}

var (
	_                                WithdrawalMessage = (*LegacyWithdrawal)(nil)
	relayMessageSelector                               = crypto.Keccak256([]byte("relayMessage(address,address,bytes,uint256)"))[0:4]
	ErrUnexpectedMessagePasserCaller                   = errors.New("unexpected message passer caller")
)

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
	enc, err := EncodeCrossDomainMessageV0(w.Target, w.Sender, []byte(w.Data), w.Nonce)
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

	// Check to make sure that the selector matches
	if !bytes.Equal(data[0:4], relayMessageSelector) {
		return fmt.Errorf("invalid selector: 0x%x", data[0:4])
	}

	msgSender := data[len(data)-len(predeploys.L2CrossDomainMessengerAddr):]
	w.MessageSender = common.BytesToAddress(msgSender)

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
	w.Data = hexutil.Bytes(msgData)
	w.Nonce = nonce
	return nil
}

// Check will error if the MessageSender is not the L2CrossDomainMessengerAddr.
// This is important during the withdrawal process as only calls from the
// L2CrossDomainMessenger to the LegacyMessagePasser should be considered for
// migration.
func (w *LegacyWithdrawal) Check() error {
	if w.MessageSender != predeploys.L2CrossDomainMessengerAddr {
		return ErrUnexpectedMessagePasserCaller
	}
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
	method, err := abi.MethodById(w.Data)
	// If it is an unknown selector, there is no value
	if err != nil {
		return value, nil
	}

	if w.Sender == nil {
		return nil, errors.New("sender is nil")
	}

	isFromL2StandardBridge := *w.Sender == predeploys.L2StandardBridgeAddr
	if isFromL2StandardBridge && method.Name == "finalizeETHWithdrawal" {
		data, err := method.Inputs.Unpack(w.Data[4:])
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
		Nonce:    w.Nonce,
		Sender:   w.Sender,
		Target:   w.Target,
		Value:    new(big.Int),
		GasLimit: new(big.Int),
		Data:     []byte(w.Data),
	}
}
