package eth

import (
	"crypto/sha256"
	"encoding/binary"
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

// FromData encodes the given input data into this blob. The encoding scheme is as follows:
//
// First, field elements are encoded as big-endian uint256 in BLS modulus range. To avoid modulus
// overflow, we can't use the full 32 bytes, so we write data only to the topmost 31 bytes of each.
// TODO: we can optimize this to get a bit more data from the blobs by using the top byte
// partially.
//
// The first field element encodes the length of input data as a little endian uint32 in its
// topmost 4 (out of 31) bytes, and the first 27 bytes of the input data in its remaining 27
// bytes.
//
// The remaining field elements each encode 31 bytes of the remaining input data, up until the end
// of the input.
//
// TODO: version the encoding format to allow for future encoding changes
func (b *Blob) FromData(data Data) error {
	if len(data) > MaxBlobDataSize {
		return fmt.Errorf("data is too large for blob. len=%v", len(data))
	}
	b.Clear()
	// encode 4-byte little-endian length value into topmost 4 bytes (out of 31) of first field
	// element
	binary.LittleEndian.PutUint32(b[1:5], uint32(len(data)))
	// encode first 27 bytes of input data into remaining bytes of first field element
	offset := copy(b[5:32], data)
	// encode (up to) 31 bytes of remaining input data at a time into the subsequent field element
	for i := 1; i < 4096; i++ {
		offset += copy(b[i*32+1:i*32+32], data[offset:])
		if offset == len(data) {
			break
		}
	}
	if offset < len(data) {
		return fmt.Errorf("failed to fit all data into blob. bytes remaining: %v", len(data)-offset)
	}
	return nil
}

// ToData decodes the blob into raw byte data. See FromData above for details on the encoding
// format.
func (b *Blob) ToData() (Data, error) {
	data := make(Data, 4096*32)
	for i := 0; i < 4096; i++ {
		if b[i*32] != 0 {
			return nil, fmt.Errorf("invalid blob, found non-zero high order byte %x of field element %d", b[i*32], i)
		}
		copy(data[i*31:i*31+31], b[i*32+1:i*32+32])
	}
	// extract the length prefix & trim the output accordingly
	dataLen := binary.LittleEndian.Uint32(data[:4])
	data = data[4:]
	if dataLen > uint32(len(data)) {
		return nil, fmt.Errorf("invalid blob, length prefix out of range: %d", dataLen)
	}
	data = data[:dataLen]
	return data, nil
}

func (b *Blob) Clear() {
	for i := 0; i < BlobSize; i++ {
		b[i] = 0
	}
}
