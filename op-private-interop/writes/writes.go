// Package writes defines public, opaque records of private state changes.
package writes

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Each record contains a state identifier and a commitment to its resulting value.
const RecordSize = 72
const MaxEncodedSize = 4 * 1024 * 1024

// Kinds separate account fields from storage slots. A slot dependency must also
// track Existence and StorageReset for its account.
const (
	Existence byte = iota
	Balance
	Nonce
	CodeHash
	StorageReset
	Storage
)

var ErrInvalid = errors.New("invalid private write records")

// Record never carries a plaintext address, storage key, or value.
type Record struct {
	Tag             common.Hash `json:"tag"`
	ValueCommitment common.Hash `json:"valueCommitment"`
	BlockNumber     uint64      `json:"blockNumber"`
}

// Tag uses fixed-width fields and chain separation. Hashing hides only identifiers
// an observer cannot guess; this is not encryption.
func Tag(chainID uint64, kind byte, address common.Address, slot common.Hash) common.Hash {
	var chain [8]byte
	binary.BigEndian.PutUint64(chain[:], chainID)
	return crypto.Keccak256Hash([]byte("optimism.private-interop.write-key.v1"), chain[:], []byte{kind}, address[:], slot[:])
}

// Commit binds a 32-byte canonical value to its state identifier.
func Commit(tag, value common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte("optimism.private-interop.write-value.v1"), tag[:], value[:])
}

// Encode requires strictly sorted, unique tags. Empty is an explicit no-change record.
func Encode(records []Record) ([]byte, error) {
	if len(records) > MaxEncodedSize/RecordSize {
		return nil, ErrInvalid
	}
	out := make([]byte, len(records)*RecordSize)
	for i, r := range records {
		if i > 0 && bytes.Compare(records[i-1].Tag[:], r.Tag[:]) >= 0 {
			return nil, ErrInvalid
		}
		copy(out[i*RecordSize:], r.Tag[:])
		copy(out[i*RecordSize+32:], r.ValueCommitment[:])
		binary.BigEndian.PutUint64(out[i*RecordSize+64:], r.BlockNumber)
	}
	return out, nil
}

func Decode(data []byte) ([]Record, error) {
	if len(data) > MaxEncodedSize || len(data)%RecordSize != 0 {
		return nil, ErrInvalid
	}
	var out []Record
	for i := 0; i < len(data); i += RecordSize {
		r := Record{Tag: common.BytesToHash(data[i : i+32]), ValueCommitment: common.BytesToHash(data[i+32 : i+64]), BlockNumber: binary.BigEndian.Uint64(data[i+64 : i+72])}
		if len(out) > 0 && bytes.Compare(out[len(out)-1].Tag[:], r.Tag[:]) >= 0 {
			return nil, ErrInvalid
		}
		out = append(out, r)
	}
	return out, nil
}

// Accumulator retains the last commitment for every key changed in a range.
// Apply must be called in block order. A key is retained even if a later block
// restores its value: this deliberately reports a conservative conflict.
type Accumulator map[common.Hash]Record

func (a Accumulator) Apply(records []Record) error {
	if _, err := Encode(records); err != nil {
		return err
	}
	count := len(a)
	for _, r := range records {
		old, ok := a[r.Tag]
		if !ok {
			count++
		} else if old.BlockNumber > r.BlockNumber || (old.BlockNumber == r.BlockNumber && old != r) {
			return fmt.Errorf("%w: unordered block writes", ErrInvalid)
		}
	}
	if count > MaxEncodedSize/RecordSize {
		return fmt.Errorf("%w: range too large", ErrInvalid)
	}
	for _, r := range records {
		a[r.Tag] = r
	}
	return nil
}

func (a Accumulator) Records() []Record {
	out := make([]Record, 0, len(a))
	for _, r := range a {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i].Tag[:], out[j].Tag[:]) < 0 })
	return out
}
