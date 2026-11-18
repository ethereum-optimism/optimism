package types

import (
	"bytes"
	"errors"
	"fmt"
)

// PostExecTxType is the EIP-2718 type byte of OP Stack post-execution transactions.
const PostExecTxType = 0x7D

// PostExecTx is a synthetic, unsigned OP Stack transaction used to carry
// post-execution metadata in SDM blocks. Its canonical encoding is
// PostExecTxType || Data, where Data is appended verbatim with no outer RLP
// envelope, matching op-geth's types.PostExecTx wire format. Data is itself an
// RLP-encoded payload, but op-geth (and this type) treat it as opaque bytes and
// never parse it; use [DecodePostExecPayload] to decode Data. NB: the Rust side
// (op-alloy TxPostExec) parses and validates the payload on decode — malformed
// payload bytes are accepted here but rejected there, a Go-lenient/Rust-strict
// wire-acceptance divergence inherited from op-geth's design.
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

// IsPostExecTxBytes reports whether raw, a transaction's EIP-2718 encoding, starts with the
// post-exec type byte. Empty input is not a post-exec transaction. It does not check the payload.
func IsPostExecTxBytes(raw []byte) bool {
	return len(raw) > 0 && raw[0] == PostExecTxType
}
