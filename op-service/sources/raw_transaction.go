package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// RawTransaction is a canonical binary-encoded transaction — a typed-tx
// envelope, or the RLP of a legacy transaction — that (un)marshals the
// eth_getBlockBy* JSON transaction-object form. The OP Stack synthetic classes
// (0x7E deposit, 0x7D post-exec) are codec'd by op-core/types; everything else
// by go-ethereum. Keeping block transactions in this form lets an L2 block
// round-trip without go-ethereum's types.Transaction ever holding an OP
// synthetic transaction: the transactions-trie leaf of a typed transaction IS
// its opaque encoding, and class-specific access decodes on demand.
type RawTransaction hexutil.Bytes

// Type returns the EIP-2718 transaction type byte, or types.LegacyTxType for
// legacy (RLP-list) payloads and empty input.
func (tx RawTransaction) Type() byte {
	if len(tx) == 0 || tx[0] > 0x7f {
		return types.LegacyTxType
	}
	return tx[0]
}

// Hash returns the transaction hash: the keccak-256 hash of the canonical
// encoding — legacy RLP, or the EIP-2718 typed-tx envelope.
func (tx RawTransaction) Hash() common.Hash {
	return crypto.Keccak256Hash(tx)
}

// UnmarshalJSON decodes an RPC transaction object into its canonical binary
// encoding, routing by the EIP-2718 type field. A JSON null decodes to an
// empty RawTransaction, which block verification rejects.
func (tx *RawTransaction) UnmarshalJSON(input []byte) error {
	if string(input) == "null" {
		*tx = nil
		return nil
	}
	var peek struct {
		Type *hexutil.Uint64 `json:"type"`
	}
	if err := json.Unmarshal(input, &peek); err != nil {
		return err
	}
	var typ uint64 // an absent type field means a legacy transaction
	if peek.Type != nil {
		typ = uint64(*peek.Type)
	}
	var raw []byte
	var err error
	// Dispatch on the full type value: a value with a matching low byte but
	// out of byte range (e.g. 0x17e) must not route to an OP codec — the
	// default arm rejects it like op-geth does.
	switch typ {
	case uint64(optypes.DepositTxType):
		var d optypes.DepositTx
		if err := json.Unmarshal(input, &d); err != nil {
			return fmt.Errorf("invalid deposit tx: %w", err)
		}
		raw, err = d.MarshalBinary()
	case uint64(optypes.PostExecTxType):
		var p optypes.PostExecTx
		if err := json.Unmarshal(input, &p); err != nil {
			return fmt.Errorf("invalid post-exec tx: %w", err)
		}
		raw, err = p.MarshalBinary()
	default:
		var gtx types.Transaction
		if err := gtx.UnmarshalJSON(input); err != nil {
			return err
		}
		raw, err = gtx.MarshalBinary()
	}
	if err != nil {
		return fmt.Errorf("failed to encode tx type %#x: %w", typ, err)
	}
	*tx = raw
	return nil
}

// MarshalJSON encodes the transaction as an RPC transaction object,
// the inverse of UnmarshalJSON.
func (tx RawTransaction) MarshalJSON() ([]byte, error) {
	if len(tx) == 0 {
		return nil, errors.New("empty transaction bytes")
	}
	switch tx.Type() {
	case optypes.DepositTxType:
		d, err := optypes.UnmarshalDepositTx(tx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(d)
	case optypes.PostExecTxType:
		p, err := optypes.UnmarshalPostExecTx(tx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(p)
	default:
		var gtx types.Transaction
		if err := gtx.UnmarshalBinary(tx); err != nil {
			return nil, err
		}
		return gtx.MarshalJSON()
	}
}

// RawTransactions is a list of canonical binary-encoded transactions. It
// implements types.DerivableList: the transactions-trie leaf of a typed
// transaction is its opaque encoding verbatim, and a legacy transaction's raw
// form already is its RLP.
type RawTransactions []RawTransaction

var _ types.DerivableList = (RawTransactions)(nil)

func (txs RawTransactions) Len() int { return len(txs) }

func (txs RawTransactions) EncodeIndex(i int, w *bytes.Buffer) {
	w.Write(txs[i])
}

// Hashes returns the transaction hashes.
func (txs RawTransactions) Hashes() []common.Hash {
	out := make([]common.Hash, len(txs))
	for i, tx := range txs {
		out[i] = tx.Hash()
	}
	return out
}

// Geth decodes all transactions into go-ethereum types.Transactions.
//
// Use the class-partitioned accessors instead for L2 blocks: they can contain
// OP Stack synthetic transactions (0x7E deposits, 0x7D post-exec), which
// upstream go-ethereum's transaction decoding rejects. This decode works on
// such blocks only while the build resolves go-ethereum to op-geth.
func (txs RawTransactions) Geth() (types.Transactions, error) {
	out := make(types.Transactions, len(txs))
	for i, raw := range txs {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(raw); err != nil {
			return nil, fmt.Errorf("failed to decode tx %d: %w", i, err)
		}
		out[i] = tx
	}
	return out, nil
}

// UserTxs decodes the standard Ethereum transactions of the list — every
// transaction that is not an OP Stack synthetic (0x7E deposit or 0x7D
// post-exec) — into go-ethereum types.Transactions.
func (txs RawTransactions) UserTxs() (types.Transactions, error) {
	out := make(types.Transactions, 0, len(txs))
	for i, raw := range txs {
		if len(raw) == 0 {
			return nil, fmt.Errorf("tx %d is empty", i)
		}
		switch raw.Type() {
		case optypes.DepositTxType, optypes.PostExecTxType:
			continue
		}
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(raw); err != nil {
			return nil, fmt.Errorf("failed to decode tx %d: %w", i, err)
		}
		out = append(out, tx)
	}
	return out, nil
}

// Deposits decodes the deposit (0x7E) transactions of the list.
func (txs RawTransactions) Deposits() ([]*optypes.DepositTx, error) {
	var out []*optypes.DepositTx
	for i, raw := range txs {
		if len(raw) == 0 {
			return nil, fmt.Errorf("tx %d is empty", i)
		}
		if raw.Type() != optypes.DepositTxType {
			continue
		}
		d, err := optypes.UnmarshalDepositTx(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to decode deposit tx %d: %w", i, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// ErrMissingFirstDeposit is returned by first-deposit accessors when a block's
// first transaction is not an OP Stack deposit (empty blocks, L1 blocks).
var ErrMissingFirstDeposit = errors.New("block's first transaction is not a deposit")

// FirstDeposit decodes the first transaction of the list as a deposit (0x7E).
// It returns ErrMissingFirstDeposit if the list is empty or the first
// transaction is of another type.
func (txs RawTransactions) FirstDeposit() (*optypes.DepositTx, error) {
	if len(txs) == 0 || txs[0].Type() != optypes.DepositTxType {
		return nil, ErrMissingFirstDeposit
	}
	return optypes.UnmarshalDepositTx(txs[0])
}
