package types

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

// PostExecTxType is the EIP-2718 type byte of OP Stack post-execution transactions.
const PostExecTxType = byte(0x7D)

// PostExecTx is a synthetic, unsigned OP Stack transaction used to carry
// post-execution metadata in SDM blocks. Its canonical encoding is
// PostExecTxType || Data, where Data is appended verbatim with no outer RLP
// envelope, matching op-geth's types.PostExecTx wire format. Data is itself an
// RLP-encoded payload, but op-geth (and this type) treat it as opaque bytes and
// never parse it.
type PostExecTx struct {
	Data []byte
}

// MarshalBinary returns the canonical EIP-2718 encoding of the post-exec transaction.
func (p *PostExecTx) MarshalBinary() ([]byte, error) {
	out := make([]byte, 0, 1+len(p.Data))
	out = append(out, PostExecTxType)
	return append(out, p.Data...), nil
}

// UnmarshalPostExecTx decodes a post-exec transaction from its EIP-2718 encoding.
// Like op-geth, it rejects an empty payload.
func UnmarshalPostExecTx(raw []byte) (*PostExecTx, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty transaction bytes")
	}
	if raw[0] != PostExecTxType {
		return nil, fmt.Errorf("expected post-exec tx type byte %#x, got %#x", PostExecTxType, raw[0])
	}
	if len(raw) == 1 {
		return nil, errors.New("post-exec tx payload is empty")
	}
	return &PostExecTx{Data: bytes.Clone(raw[1:])}, nil
}

// IsPostExecTx reports whether tx is an OP Stack post-execution transaction.
//
// Like the deposit helpers in deposit_tx.go, this is transition-only: a
// go-ethereum *types.Transaction can hold a post-exec tx only while the build
// resolves to op-geth. After the cutover to upstream go-ethereum the 0x7D type
// is rejected on decode, so this is removed; the durable shape decodes raw
// bytes via [UnmarshalPostExecTx].
func IsPostExecTx(tx *types.Transaction) bool {
	return tx.Type() == PostExecTxType
}
