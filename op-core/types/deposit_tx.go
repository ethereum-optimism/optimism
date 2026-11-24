// Package types holds the OP-Stack-specific transaction and receipt types.
// Consumers import it as optypes.
package types

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// DepositTxType is the EIP-2718 type byte of OP Stack deposit transactions.
const DepositTxType = 0x7E

// DepositTx is an OP Stack deposit transaction, derived from L1 (or generated
// for network upgrades) rather than signed by a user. Its canonical encoding is
// DepositTxType || RLP(fields), matching op-geth's types.DepositTx wire format.
type DepositTx struct {
	// SourceHash uniquely identifies the source of the deposit.
	SourceHash common.Hash
	// From is the sender address, determined by the deposit's origin instead of a signature.
	From common.Address
	// To is the recipient; nil means contract creation.
	To *common.Address `rlp:"nil"`
	// Mint is minted on L2, locked on L1; nil if no minting. Note that nil and
	// zero share the same wire encoding and decode to zero.
	Mint *big.Int `rlp:"nil"`
	// Value is transferred from the L2 balance, executed after Mint (if any).
	Value *big.Int
	// Gas is the gas limit.
	Gas uint64
	// IsSystemTransaction indicates the transaction is exempt from the L2 gas limit.
	IsSystemTransaction bool
	// Data is the calldata.
	Data []byte
}

// MarshalBinary returns the canonical EIP-2718 encoding of the deposit transaction.
func (d *DepositTx) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(DepositTxType)
	if err := rlp.Encode(&buf, d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Hash returns the transaction hash: the keccak-256 hash of the canonical
// EIP-2718 encoding. A well-formed deposit cannot fail to encode, so an encoding
// error is unreachable and panics; this keeps the signature free of an error
// return, matching how a transaction hash is used at call sites.
func (d *DepositTx) Hash() common.Hash {
	raw, err := d.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(raw)
}

// UnmarshalDepositTx decodes a deposit transaction from its EIP-2718 encoding.
func UnmarshalDepositTx(raw []byte) (*DepositTx, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty transaction bytes")
	}
	if raw[0] != DepositTxType {
		return nil, fmt.Errorf("expected deposit tx type byte %#x, got %#x", DepositTxType, raw[0])
	}
	d := new(DepositTx)
	if err := rlp.DecodeBytes(raw[1:], d); err != nil {
		return nil, fmt.Errorf("invalid deposit tx payload: %w", err)
	}
	return d, nil
}

// IsDepositTxBytes reports whether raw, a transaction's EIP-2718 encoding, starts with the
// deposit type byte. Empty input is not a deposit. It does not check the payload.
func IsDepositTxBytes(raw []byte) bool {
	return len(raw) > 0 && raw[0] == DepositTxType
}
