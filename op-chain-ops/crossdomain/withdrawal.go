package crossdomain

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ WithdrawalMessage = (*Withdrawal)(nil)

// Withdrawal represents a withdrawal transaction on L2
type Withdrawal struct {
	Nonce    *big.Int        `json:"nonce"`
	Sender   *common.Address `json:"sender"`
	Target   *common.Address `json:"target"`
	Value    *big.Int        `json:"value"`
	GasLimit *big.Int        `json:"gasLimit"`
	Data     []byte          `json:"data"`
}

// NewWithdrawal will create a Withdrawal
func NewWithdrawal(
	nonce *big.Int,
	sender, target *common.Address,
	value, gasLimit *big.Int,
	data []byte,
) *Withdrawal {
	return &Withdrawal{
		Nonce:    nonce,
		Sender:   sender,
		Target:   target,
		Value:    value,
		GasLimit: gasLimit,
		Data:     data,
	}
}

// Encode will serialize the Withdrawal so that it is suitable for hashing.
func (w *Withdrawal) Encode() ([]byte, error) {
	args := abi.Arguments{
		{Name: "nonce", Type: Uint256Type},
		{Name: "sender", Type: AddressType},
		{Name: "target", Type: AddressType},
		{Name: "value", Type: Uint256Type},
		{Name: "gasLimit", Type: Uint256Type},
		{Name: "data", Type: BytesType},
	}
	enc, err := args.Pack(w.Nonce, w.Sender, w.Target, w.Value, w.GasLimit, w.Data)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// Decode will deserialize a Withdrawal
func (w *Withdrawal) Decode(data []byte) error {
	args := abi.Arguments{
		{Name: "nonce", Type: Uint256Type},
		{Name: "sender", Type: AddressType},
		{Name: "target", Type: AddressType},
		{Name: "value", Type: Uint256Type},
		{Name: "gasLimit", Type: Uint256Type},
		{Name: "data", Type: BytesType},
	}
	decoded, err := args.Unpack(data)
	if err != nil {
		return err
	}

	nonce, ok := decoded[0].(*big.Int)
	if !ok {
		return errors.New("cannot abi decode nonce")
	}
	sender, ok := decoded[1].(common.Address)
	if !ok {
		return errors.New("cannot abi decode sender")
	}
	target, ok := decoded[2].(common.Address)
	if !ok {
		return errors.New("cannot abi decode target")
	}
	value, ok := decoded[3].(*big.Int)
	if !ok {
		return errors.New("cannot abi decode value")
	}
	gasLimit, ok := decoded[4].(*big.Int)
	if !ok {
		return errors.New("cannot abi decode gasLimit")
	}
	msgData, ok := decoded[5].([]byte)
	if !ok {
		return errors.New("cannot abi decode data")
	}

	w.Nonce = nonce
	w.Sender = &sender
	w.Target = &target
	w.Value = value
	w.GasLimit = gasLimit
	w.Data = msgData
	return nil
}

// Hash will hash the Withdrawal. This is the hash that is computed in
// the L2ToL1MessagePasser. The encoding is the same as the v1 cross domain
// message encoding without the 4byte selector prepended.
func (w *Withdrawal) Hash() (common.Hash, error) {
	encoded, err := w.Encode()
	if err != nil {
		return common.Hash{}, err
	}
	hash := crypto.Keccak256(encoded)
	return common.BytesToHash(hash), nil
}

// StorageSlot will compute the storage slot that will be set to
// true in the L2ToL1MessagePasser. The withdrawal proof sent to
// L1 will prove that this storage slot is set to "true".
func (w *Withdrawal) StorageSlot() (common.Hash, error) {
	hash, err := w.Hash()
	if err != nil {
		return common.Hash{}, err
	}
	preimage := make([]byte, 64)
	copy(preimage, hash.Bytes())

	slot := crypto.Keccak256(preimage)
	return common.BytesToHash(slot), nil
}
