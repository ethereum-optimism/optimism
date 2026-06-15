// Package types holds the OP-Stack-specific transaction and receipt types.
// Consumers import it as optypes.
package types

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// DepositTxType is the EIP-2718 type byte of OP Stack deposit transactions.
const DepositTxType = byte(0x7E)

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

// The helpers below operate on a go-ethereum *types.Transaction. They exist
// only for the decoupling transition: while the build still resolves
// go-ethereum to op-geth, a *types.Transaction can hold a deposit, so these let
// call sites drop their dependency on op-geth's Transaction methods before the
// underlying transaction representation is migrated. Once the build moves to
// upstream go-ethereum a *types.Transaction can never hold a deposit (the
// 0x7E type is rejected on decode), making these dead — they are removed at the
// cutover, together with the differential test. The durable shape decodes raw
// bytes into a [DepositTx] (via [UnmarshalDepositTx]) and reads its fields
// directly; see op-service/sources accessors (#20264).

// IsDepositTx reports whether tx is an OP Stack deposit transaction.
func IsDepositTx(tx *types.Transaction) bool {
	return tx.Type() == DepositTxType
}

// SourceHash returns the source hash of a deposit transaction.
// It errors if tx is not a deposit transaction.
func SourceHash(tx *types.Transaction) (common.Hash, error) {
	d, err := asDepositTx(tx)
	if err != nil {
		return common.Hash{}, err
	}
	return d.SourceHash, nil
}

// Mint returns the ETH minted by a deposit transaction. A deposit that mints
// nothing yields zero, never nil: the wire encoding does not distinguish a nil
// from a zero mint, and the transaction is decoded from its wire bytes.
// It errors if tx is not a deposit transaction.
func Mint(tx *types.Transaction) (*big.Int, error) {
	d, err := asDepositTx(tx)
	if err != nil {
		return nil, err
	}
	return d.Mint, nil
}

// IsSystemTx reports whether tx is a deposit that is a system transaction,
// exempt from the L2 gas limit. It errors if tx is not a deposit transaction.
func IsSystemTx(tx *types.Transaction) (bool, error) {
	d, err := asDepositTx(tx)
	if err != nil {
		return false, err
	}
	return d.IsSystemTransaction, nil
}

func asDepositTx(tx *types.Transaction) (*DepositTx, error) {
	if !IsDepositTx(tx) {
		return nil, fmt.Errorf("transaction %s (type %#x) is not a deposit transaction", tx.Hash(), tx.Type())
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction %s: %w", tx.Hash(), err)
	}
	return UnmarshalDepositTx(raw)
}
