package eth

import (
	"crypto/sha256"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

const (
	BlobSize        = 4096 * 32
	MaxBlobDataSize = 4096*31 - 4
)

type Blob [BlobSize]byte

func (b *Blob) KZGBlob() *kzg4844.Blob {
	return (*kzg4844.Blob)(b)
}

func (b *Blob) UnmarshalJSON(text []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Blob) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("Bytes32", text, b[:])
}

func (b *Blob) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b *Blob) String() string {
	return hexutil.Encode(b[:])
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b *Blob) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[BlobSize-3:])
}

func (b *Blob) ComputeKZGCommitment() (kzg4844.Commitment, error) {
	return kzg4844.BlobToCommitment(*b.KZGBlob())
}

// KZGToVersionedHash computes the "blob hash" (a.k.a. versioned-hash) of a blob-commitment, as used in a blob-tx.
// We implement it here because it is unfortunately not (currently) exposed by geth.
func KZGToVersionedHash(commitment kzg4844.Commitment) (out common.Hash) {
	// EIP-4844 spec:
	//	def kzg_to_versioned_hash(commitment: KZGCommitment) -> VersionedHash:
	//		return VERSIONED_HASH_VERSION_KZG + sha256(commitment)[1:]
	h := sha256.New()
	h.Write(commitment[:])
	_ = h.Sum(out[:0])
	out[0] = params.BlobTxHashVersion
	return out
}

// VerifyBlobProof verifies that the given blob and proof corresponds to the given commitment,
// returning error if the verification fails.
func VerifyBlobProof(blob *Blob, commitment kzg4844.Commitment, proof kzg4844.Proof) error {
	return kzg4844.VerifyBlobProof(*blob.KZGBlob(), commitment, proof)
}
