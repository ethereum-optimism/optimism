package types

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Receipts implements types.DerivableList so the receipts root of an OP Stack
// block can be recomputed with types.DeriveSha. The OP Stack synthetic receipt
// types are encoded by this package; everything else delegates to go-ethereum.
//
// The encoding reads the OUTER Receipt fields (DepositNonce,
// DepositReceiptVersion) — they are authoritative, per the Receipt contract.
// Receipts must be produced through this package's decoders; a wrapper
// hand-built around a receipt whose deposit fields live only on the embedded
// struct encodes without them.
type Receipts []*Receipt

var _ types.DerivableList = (Receipts)(nil)

// Geth returns a view of the embedded go-ethereum receipts, for boundaries
// that operate on standard receipt fields only. The elements alias the
// receivers' embedded structs — mutations are visible in both.
func (rs Receipts) Geth() types.Receipts {
	out := make(types.Receipts, len(rs))
	for i, r := range rs {
		out[i] = &r.Receipt
	}
	return out
}

// Len returns the number of receipts in the list.
func (rs Receipts) Len() int { return len(rs) }

// receiptRLP is the consensus encoding of a receipt.
type receiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
}

// depositReceiptRLP is the consensus encoding of a deposit (0x7E) receipt.
// DepositNonce was introduced in Regolith; DepositReceiptVersion in Canyon,
// indicating that the receipt hash preimage includes the nonce and version.
type depositReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
	DepositNonce      *uint64 `rlp:"optional"`
	// DepositReceiptVersion is non-nil post-Canyon; the receipt hash
	// post-Regolith but pre-Canyon inadvertently did not include DepositNonce.
	DepositReceiptVersion *uint64 `rlp:"optional"`
}

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
)

func (r *Receipt) statusEncoding() []byte {
	if len(r.PostState) > 0 {
		return r.PostState
	}
	if r.Status == types.ReceiptStatusFailed {
		return receiptStatusFailedRLP
	}
	return receiptStatusSuccessfulRLP
}

func (r *Receipt) setStatus(postStateOrStatus []byte) error {
	switch {
	case bytes.Equal(postStateOrStatus, receiptStatusSuccessfulRLP):
		r.Status = types.ReceiptStatusSuccessful
	case bytes.Equal(postStateOrStatus, receiptStatusFailedRLP):
		r.Status = types.ReceiptStatusFailed
	case len(postStateOrStatus) == common.HashLength:
		r.PostState = postStateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", postStateOrStatus)
	}
	return nil
}

// EncodeIndex encodes the i'th receipt to w, for receipts-root derivation.
//
// The deposit arms must not be changed to delegate to a receipt's MarshalBinary:
// for a deposit receipt with non-nil DepositNonce but nil DepositReceiptVersion
// (post-Regolith, pre-Canyon), the hash preimage deliberately EXCLUDES the nonce
// while MarshalBinary includes it — preserving backwards compatibility of the
// receipts-root computation, matching op-geth.
func (rs Receipts) EncodeIndex(i int, w *bytes.Buffer) {
	r := rs[i]
	switch r.Type {
	case DepositTxType:
		w.WriteByte(DepositTxType)
		if r.DepositReceiptVersion != nil {
			// post-Canyon receipt hash computation includes nonce and version
			_ = rlp.Encode(w, &depositReceiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs, r.DepositNonce, r.DepositReceiptVersion})
		} else {
			_ = rlp.Encode(w, &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs})
		}
	case PostExecTxType:
		w.WriteByte(PostExecTxType)
		_ = rlp.Encode(w, &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs})
	default:
		types.Receipts{&r.Receipt}.EncodeIndex(0, w)
	}
}

// MarshalBinary returns the consensus encoding of the receipt, routing the
// OP Stack synthetic receipt types (0x7E deposit, 0x7D post-exec) to this
// package and everything else to go-ethereum. The deposit encoding reads the
// authoritative OUTER DepositNonce/DepositReceiptVersion fields and — unlike
// the receipts-root derivation in EncodeIndex — always includes the nonce when
// set, matching op-geth's MarshalBinary. Inverse of UnmarshalBinary.
func (r *Receipt) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	switch r.Type {
	case DepositTxType:
		buf.WriteByte(DepositTxType)
		err := rlp.Encode(&buf, &depositReceiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs, r.DepositNonce, r.DepositReceiptVersion})
		return buf.Bytes(), err
	case PostExecTxType:
		buf.WriteByte(PostExecTxType)
		err := rlp.Encode(&buf, &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs})
		return buf.Bytes(), err
	default:
		return r.Receipt.MarshalBinary()
	}
}

// UnmarshalBinary decodes the consensus encoding of a receipt, routing the
// OP Stack synthetic receipt types (0x7E deposit, 0x7D post-exec) to this
// package and everything else to go-ethereum. The OP JSON-only fee fields
// (L1GasPrice, OperatorFeeScalar, ...) are not part of the consensus encoding
// and remain nil.
func (r *Receipt) UnmarshalBinary(b []byte) error {
	if len(b) <= 1 {
		// Legacy receipts are RLP lists (first byte >= 0xc0), so a valid
		// 1-byte input does not exist; delegate for the canonical error.
		return r.Receipt.UnmarshalBinary(b)
	}
	switch b[0] {
	case DepositTxType:
		var data depositReceiptRLP
		if err := rlp.DecodeBytes(b[1:], &data); err != nil {
			return err
		}
		r.Type = DepositTxType
		r.DepositNonce = data.DepositNonce
		r.DepositReceiptVersion = data.DepositReceiptVersion
		// While go-ethereum resolves to op-geth, the embedded receipt carries
		// identically-named shadow fields; keep them consistent so op-geth code
		// reading through Geth() sees the same values. These two assignments
		// stop compiling at the final cutover and are removed then.
		r.Receipt.DepositNonce = data.DepositNonce
		r.Receipt.DepositReceiptVersion = data.DepositReceiptVersion
		r.CumulativeGasUsed, r.Bloom, r.Logs = data.CumulativeGasUsed, data.Bloom, data.Logs
		return r.setStatus(data.PostStateOrStatus)
	case PostExecTxType:
		var data receiptRLP
		if err := rlp.DecodeBytes(b[1:], &data); err != nil {
			return err
		}
		r.Type = PostExecTxType
		r.CumulativeGasUsed, r.Bloom, r.Logs = data.CumulativeGasUsed, data.Bloom, data.Logs
		return r.setStatus(data.PostStateOrStatus)
	default:
		return r.Receipt.UnmarshalBinary(b)
	}
}
