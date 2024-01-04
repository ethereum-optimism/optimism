package eth

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

const (
	BlobSize        = 4096 * 32
	MaxBlobDataSize = (4*31+3)*1024 - 4
	EncodingVersion = 0
	VersionOffset   = 1    // offset of the version byte in the blob encoding
	Rounds          = 1024 // number of encode/decode rounds
)

var (
	ErrBlobInvalidFieldElement    = errors.New("invalid field element")
	ErrBlobInvalidEncodingVersion = errors.New("invalid encoding version")
	ErrBlobInvalidLength          = errors.New("invalid length for blob")
	ErrBlobInputTooLarge          = errors.New("too much data to encode in one blob")
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
// In each round we perform 7 reads of input of lengths (31,1,31,1,31,1,31) bytes respectively for
// a total of 127 bytes. This data is encoded into the next 4 field elements of the output by
// placing each of the 4x31 byte chunks into bytes [1:32] of its respective field element. The
// three single byte chunks (24 bits) are split into 4x6-bit chunks, each of which is written into
// the top most byte of its respective field element, leaving the top 2 bits of each field element
// empty to avoid modulus overflow.  This process is repeated for up to 1024 rounds until all data
// is encoded.
//
// For only the very first output field, bytes [1:5] are used to encode the version and the length
// of the data.
func (b *Blob) FromData(data Data) error {
	if len(data) > MaxBlobDataSize {
		return fmt.Errorf("%w: len=%v", ErrBlobInputTooLarge, data)
	}
	b.Clear()

	// first field element encodes the version and the length of the data in [1:5]
	b[VersionOffset] = EncodingVersion

	// Encode the length as big-endian uint24 into [2:5] bytes of the first field element.
	// The length check above ensures we can always fit the length value into only 3 bytes.
	ilen := uint32(len(data))
	b[2] = byte(ilen >> 16)
	b[3] = byte(ilen >> 8)
	b[4] = byte(ilen)

	// round 0 is special cased here since we can only encode the first (31-5)+1=27 bytes of input
	// into the first field element due to version/length encoding already occupying the first 5
	// bytes.
	ipos := 0 // current position of the input data from where we are reading
	buffer, ipos := read(ipos, 27, data)
	x, ipos := read(ipos, 1, data)
	encodedByte := x[0] & 0b0011_1111
	b[0] = encodedByte
	b.write(buffer, 5)
	opos := 32 // current position of the blob where we are writing

	buffer, ipos = read(ipos, 31, data)
	y, ipos := read(ipos, 1, data)

	encodedByte = (y[0] & 0b0000_1111) | ((x[0] & 0b1100_0000) >> 2)
	opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

	buffer, ipos = read(ipos, 31, data)
	z, ipos := read(ipos, 1, data)
	encodedByte = z[0] & 0b0011_1111
	opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

	buffer, ipos = read(ipos, 31, data)
	encodedByte = ((z[0] & 0b1100_0000) >> 2) | ((y[0] & 0b1111_0000) >> 4)
	opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

	for round := 1; round < Rounds && ipos < len(data); round++ {
		buffer, ipos = read(ipos, 31, data)
		x, ipos = read(ipos, 1, data)
		encodedByte = x[0] & 0b0011_1111
		opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

		buffer, ipos = read(ipos, 31, data)
		y, ipos = read(ipos, 1, data)
		encodedByte = (y[0] & 0b0000_1111) | ((x[0] & 0b1100_0000) >> 2)
		opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

		buffer, ipos = read(ipos, 31, data)
		z, ipos = read(ipos, 1, data)
		encodedByte = z[0] & 0b0011_1111
		opos = b.writeByteAndBuffer(opos, encodedByte, buffer)

		buffer, ipos = read(ipos, 31, data)
		encodedByte = ((z[0] & 0b1100_0000) >> 2) | ((y[0] & 0b1111_0000) >> 4)
		opos = b.writeByteAndBuffer(opos, encodedByte, buffer)
	}

	if ipos < len(data) {
		return fmt.Errorf("failed to fit all data into blob. bytes remaining: %v", len(data)-ipos)
	}

	return nil
}

// read reads up to numBytes of input from the data starting at the given offset, and returns an
// updated offset of max(offset+numBytes, len(data)).  It always returns a byte array of length
// exactly numBytes, even if there wasn't enough data to read. In the short-read case, any trailing
// bytes are zero padded.
func read(offset int, numBytes int, data []byte) ([]byte, int) {
	byteArray := make([]byte, numBytes)
	if offset >= len(data) {
		return byteArray, len(data)
	}

	actualNumBytes := numBytes
	// actual number of bytes to read may be less than numBytes if we are near the end of the input
	if offset+numBytes > len(data) {
		actualNumBytes = len(data) - offset
	}

	copy(byteArray, data[offset:offset+actualNumBytes])

	return byteArray, offset + actualNumBytes
}

func (b *Blob) writeByteAndBuffer(opos int, encodedByte byte, buffer []byte) int {
	b[opos] = encodedByte
	opos = b.write(buffer, opos+1)
	return opos
}

func (b *Blob) write(buffer []byte, opos int) int {
	copy(b[opos:], buffer)
	return opos + len(buffer)
}

// ToData decodes the blob into raw byte data. See FromData above for details on the encoding
// format. If error is returned it will be one of InvalidFieldElementError,
// InvalidEncodingVersionError and InvalidLengthError.
func (b *Blob) ToData() (Data, error) {
	// check the version
	if b[VersionOffset] != EncodingVersion {
		return nil, fmt.Errorf(
			"%w: expected version %d, got %d", ErrBlobInvalidEncodingVersion, EncodingVersion, b[VersionOffset])
	}

	// decode the 3-byte big-endian length value into a 4-byte integer
	outputLen := uint32(b[2])<<16 | uint32(b[3])<<8 | uint32(b[4])
	if outputLen > MaxBlobDataSize {
		return nil, fmt.Errorf("%w: got %d", ErrBlobInvalidLength, outputLen)
	}

	// round 0 is special cased to copy only the remaining 27 bytes of the first field element into
	// the output due to version/length encoding already occupying its first 5 bytes.
	output := make(Data, MaxBlobDataSize)
	copy(output[0:27], b[5:])

	// now process remaining 3 field elements to complete round 0
	opos := 28 // current position into output buffer
	ipos := 32 // current position into the input blob
	var err error
	encodedByte := make([]byte, 4) // buffer for the 4 6-bit chunks
	encodedByte[0] = b[0]
	for i := 1; i < 4; i++ {
		encodedByte[i] = b[ipos]
		if opos, ipos, err = b.decodeFieldElement(opos, ipos, encodedByte, output); err != nil {
			return nil, err
		}
	}
	opos = reassembleBytes(opos, encodedByte, output)

	// in each remaining round we decode 4 field elements (128 bytes) of the input into 127 bytes
	// of output
	for i := 1; i < Rounds && opos < int(outputLen); i++ {
		for j := 0; j < 4; j++ {
			// save the first byte of each field element for later re-assembly
			encodedByte[j] = b[ipos]
			if opos, ipos, err = b.decodeFieldElement(opos, ipos, encodedByte, output); err != nil {
				return nil, err
			}
		}
		opos = reassembleBytes(opos, encodedByte, output)
	}
	output = output[:outputLen]
	return output, nil
}

// decodeFieldElement decodes the next input field element by writing its lower 31 bytes into its
// appropriate place in the output and checking the high order byte is valid. Returns an
// InvalidFieldElementError if a field element is seen with either of its two high order bits set.
func (b *Blob) decodeFieldElement(opos, ipos int, encodedByte []byte, output []byte) (int, int, error) {
	// two highest order bits of the first byte of each field element should always be 0
	if b[ipos]&(0xC0) != 0 {
		return 0, 0, fmt.Errorf("%w: field element: %d", ErrBlobInvalidFieldElement, ipos)
	}
	copy(output[opos:], b[ipos+1:ipos+32])
	return opos + 32, ipos + 32, nil
}

// reassembleBytes takes the 4x6-bit chunks from encodedByte, reassembles them into 3 bytes of
// output, and places them in their appropriate output positions.
func reassembleBytes(opos int, encodedByte []byte, output []byte) int {
	opos-- // account for fact that we don't output a 128th byte
	x := (encodedByte[0] & 0b0011_1111) | ((encodedByte[1] & 0b0011_0000) << 2)
	y := (encodedByte[1] & 0b0000_1111) | ((encodedByte[3] & 0b0000_1111) << 4)
	z := (encodedByte[2] & 0b0011_1111) | ((encodedByte[3] & 0b0011_0000) << 2)
	// put the re-assembled bytes in their appropriate output locations
	output[opos-32] = z
	output[opos-(32*2)] = y
	output[opos-(32*3)] = x
	return opos
}

func (b *Blob) Clear() {
	for i := 0; i < BlobSize; i++ {
		b[i] = 0
	}
}
