package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// txJSON is the JSON representation of OP Stack transactions on the
// eth_getBlockBy*/eth_getTransactionBy* wire. It mirrors op-geth's txJSON:
// fields without omitempty are emitted as null when unset, matching op-geth's
// encoding byte-for-byte (modulo key order). Fields of other transaction types
// are carried only to detect and reject their presence.
type txJSON struct {
	Type                 *hexutil.Uint64    `json:"type"`
	Nonce                *hexutil.Uint64    `json:"nonce"`
	To                   *common.Address    `json:"to"`
	Gas                  *hexutil.Uint64    `json:"gas"`
	GasPrice             *hexutil.Big       `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big       `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big       `json:"maxFeePerGas"`
	Value                *hexutil.Big       `json:"value"`
	Input                *hexutil.Bytes     `json:"input"`
	AccessList           *[]json.RawMessage `json:"accessList,omitempty"`
	V                    *hexutil.Big       `json:"v"`
	R                    *hexutil.Big       `json:"r"`
	S                    *hexutil.Big       `json:"s"`

	// Deposit transaction fields
	SourceHash *common.Hash    `json:"sourceHash,omitempty"`
	From       *common.Address `json:"from,omitempty"`
	Mint       *hexutil.Big    `json:"mint,omitempty"`
	IsSystemTx *bool           `json:"isSystemTx,omitempty"`

	// Only used for encoding:
	Hash common.Hash `json:"hash"`
}

func isNonZero(v *hexutil.Big) bool {
	return v != nil && v.ToInt().Sign() != 0
}

func nonZeroSignature(dec *txJSON) bool {
	return isNonZero(dec.V) || isNonZero(dec.R) || isNonZero(dec.S)
}

// MarshalJSON encodes the deposit transaction in op-geth's RPC transaction
// format, including the transaction hash. The nonce is always null: the
// effective deposit nonce that mined deposits carry on the RPC wire is not
// part of this type (or of the canonical encoding), so a decode/encode
// round-trip does not preserve it.
func (d *DepositTx) MarshalJSON() ([]byte, error) {
	typ := hexutil.Uint64(DepositTxType)
	enc := &txJSON{
		Type:       &typ,
		To:         d.To,
		Gas:        (*hexutil.Uint64)(&d.Gas),
		Value:      (*hexutil.Big)(d.Value),
		Input:      (*hexutil.Bytes)(&d.Data),
		SourceHash: &d.SourceHash,
		From:       &d.From,
		IsSystemTx: &d.IsSystemTransaction,
		Hash:       d.Hash(),
	}
	if d.Mint != nil {
		enc.Mint = (*hexutil.Big)(d.Mint)
	}
	return json.Marshal(enc)
}

// UnmarshalJSON decodes a deposit transaction from op-geth's RPC transaction
// format, applying the same field validation as op-geth. A nonce field — the
// effective deposit nonce that post-Regolith RPC responses carry — is accepted
// and ignored: it is not part of the canonical transaction encoding.
func (d *DepositTx) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Type == nil || byte(*dec.Type) != DepositTxType {
		return errors.New("transaction type is not deposit")
	}
	if dec.AccessList != nil || dec.MaxFeePerGas != nil || dec.MaxPriorityFeePerGas != nil {
		return errors.New("unexpected field(s) in deposit transaction")
	}
	if isNonZero(dec.GasPrice) {
		return errors.New("deposit transaction GasPrice must be 0")
	}
	if nonZeroSignature(&dec) {
		return errors.New("deposit transaction signature must be 0 or unset")
	}
	d.To = dec.To
	if dec.Gas == nil {
		return errors.New("missing required field 'gas' for txdata")
	}
	d.Gas = uint64(*dec.Gas)
	if dec.Value == nil {
		return errors.New("missing required field 'value' in transaction")
	}
	d.Value = (*big.Int)(dec.Value)
	// mint may be omitted or nil if there is nothing to mint.
	d.Mint = (*big.Int)(dec.Mint)
	if dec.Input == nil {
		return errors.New("missing required field 'input' in transaction")
	}
	d.Data = *dec.Input
	if dec.From == nil {
		return errors.New("missing required field 'from' in transaction")
	}
	d.From = *dec.From
	if dec.SourceHash == nil {
		return errors.New("missing required field 'sourceHash' in transaction")
	}
	d.SourceHash = *dec.SourceHash
	// IsSystemTx may be omitted. Defaults to false.
	d.IsSystemTransaction = dec.IsSystemTx != nil && *dec.IsSystemTx
	return nil
}

// Hash returns the transaction hash as op-geth computes it:
// keccak256(PostExecTxType || RLP([Data])). Note this is NOT the hash of the
// canonical envelope — the wire encoding appends Data verbatim (see
// MarshalBinary), but op-geth hashes typed transactions over the RLP of the
// inner struct, and for PostExecTx the two differ.
func (p *PostExecTx) Hash() common.Hash {
	var buf bytes.Buffer
	buf.WriteByte(PostExecTxType)
	if err := rlp.Encode(&buf, p); err != nil {
		panic(err) // a byte slice cannot fail to RLP-encode
	}
	return crypto.Keccak256Hash(buf.Bytes())
}

// MarshalJSON encodes the post-exec transaction in op-geth's RPC transaction
// format, including the transaction hash.
func (p *PostExecTx) MarshalJSON() ([]byte, error) {
	typ := hexutil.Uint64(PostExecTxType)
	zeroGas := hexutil.Uint64(0)
	enc := &txJSON{
		Type:  &typ,
		Gas:   &zeroGas,
		Value: (*hexutil.Big)(new(big.Int)),
		Input: (*hexutil.Bytes)(&p.Data),
		Hash:  p.Hash(),
	}
	return json.Marshal(enc)
}

// UnmarshalJSON decodes a post-exec transaction from op-geth's RPC transaction
// format, applying the same field validation as op-geth.
func (p *PostExecTx) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Type == nil || byte(*dec.Type) != PostExecTxType {
		return errors.New("transaction type is not post-exec")
	}
	if (dec.AccessList != nil && len(*dec.AccessList) != 0) ||
		dec.Mint != nil || dec.SourceHash != nil || dec.IsSystemTx != nil || dec.To != nil {
		return errors.New("unexpected field(s) in post-exec transaction")
	}
	if dec.From != nil && *dec.From != (common.Address{}) {
		return errors.New("post-exec transaction from must be zero address or unset")
	}
	if dec.Nonce != nil && uint64(*dec.Nonce) != 0 {
		return errors.New("post-exec transaction nonce must be 0 or unset")
	}
	if isNonZero(dec.Value) {
		return errors.New("post-exec transaction value must be 0")
	}
	if dec.Gas != nil && uint64(*dec.Gas) != 0 {
		return errors.New("post-exec transaction gas must be 0")
	}
	if nonZeroSignature(&dec) {
		return errors.New("post-exec transaction signature must be 0 or unset")
	}
	if dec.Input == nil {
		return errors.New("missing required field 'input' in transaction")
	}
	p.Data = *dec.Input
	return nil
}
