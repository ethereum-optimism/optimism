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
	MaxBlobDataSize = (4*31+3)*1024 - 4
	EncodingVersion = 0
	FieldSize       = 4 * 32   // size of a field composed of 4 field elements in bytes
	FieldCapacity   = 31*4 + 3 // # of bytes that can be encoded in 4 field elements
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
// It reads 31 bytes and then 1 byte for three times and then 31 bytes of data from the input.
// For all the 31 bytes of data, they are encoded into [1:32] bytes of each field element.
// For the extra 3 bytes read from the input, they are encoded into the 1 bytes of data on the top of each field element.
// So for each field (4 field elements), 3 bytes are encoded into 4 bytes where the highest order bit is set to 0
// This process is repeated until all data is encoded.
// For the first field, [1:5] bytes of the first field element will be used to encode the version and the length of the data.
func (b *Blob) FromData(data Data) error {
	if len(data) > MaxBlobDataSize {
		return fmt.Errorf("data is too large for blob. len=%v", len(data))
	}
	b.Clear()

	// first field element encodes the version and the length of the data in [1:5]
	b[1] = EncodingVersion

	// encode the length as big-endian uint24 into [2:5] bytes of the first field element
	if len(data) < 1<<24 {
		// Zero out any trailing data in the buffer if any
		b[2] = byte((len(data) >> 16) & 0xFF) // Most significant byte
		b[3] = byte((len(data) >> 8) & 0xFF)
		b[4] = byte(len(data) & 0xFF) // Least significant byte
	} else {
		return fmt.Errorf("Error: length_rollup_data is too large")
	}

	offset := 0
	var buffer []byte

	// encode the first 27 + 1 bytes of data into remaining bytes of first field element
	buffer, offset = read(0, 27, data)
	x, offset := read(offset, 1, data)
	encodedByte := x[0] & 0b0011_1111
	b.write([]byte{encodedByte}, 0)
	b.write(buffer, 5)
	pointer := 32 // manually set the pointer to the next field element

	clearBuffer(buffer)
	buffer, offset = read(offset, 31, data)
	y, offset := read(offset, 1, data)

	encodedByte = (y[0] & 0b0000_1111) | ((x[0] & 0b1100_0000) >> 2)
	pointer = b.write([]byte{encodedByte}, pointer)
	pointer = b.write(buffer, pointer)

	clearBuffer(buffer)
	buffer, offset = read(offset, 31, data)
	z, offset := read(offset, 1, data)
	encodedByte = z[0] & 0b0011_1111
	pointer = b.write([]byte{encodedByte}, pointer)
	pointer = b.write(buffer, pointer)

	clearBuffer(buffer)
	buffer, offset = read(offset, 31, data)
	encodedByte = ((z[0] & 0b1100_0000) >> 2) | ((y[0] & 0b1111_0000) >> 4)
	pointer = b.write([]byte{encodedByte}, pointer)
	pointer = b.write(buffer, pointer)

	if offset == len(data) {
		return nil
	}

	for fieldNumber := 1; fieldNumber < 1024; fieldNumber++ {
		clearBuffer(buffer)
		buffer, offset = read(offset, 31, data)
		x, offset = read(offset, 1, data)
		encodedByte = x[0] & 0b0011_1111
		pointer = b.write([]byte{encodedByte}, pointer)
		pointer = b.write(buffer, pointer)

		clearBuffer(buffer)
		buffer, offset = read(offset, 31, data)
		y, offset = read(offset, 1, data)
		encodedByte = (y[0] & 0b0000_1111) | ((x[0] & 0b1100_0000) >> 2)
		pointer = b.write([]byte{encodedByte}, pointer)
		pointer = b.write(buffer, pointer)

		clearBuffer(buffer)
		buffer, offset = read(offset, 31, data)
		z, offset = read(offset, 1, data)
		encodedByte = z[0] & 0b0011_1111
		pointer = b.write([]byte{encodedByte}, pointer)
		pointer = b.write(buffer, pointer)

		clearBuffer(buffer)
		buffer, offset = read(offset, 31, data)
		encodedByte = ((z[0] & 0b1100_0000) >> 2) | ((y[0] & 0b1111_0000) >> 4)
		pointer = b.write([]byte{encodedByte}, pointer)
		pointer = b.write(buffer, pointer)

		if offset >= len(data) {
			return nil
		}
	}

	if offset < len(data) {
		return fmt.Errorf("failed to fit all data into blob. bytes remaining: %v", len(data)-offset)
	}

	return nil
}

func read(offset int, numBytes int, data []byte) ([]byte, int) {
	if offset >= len(data) {
		// If the offset is at or beyond the end of data, return a new byte array of numBytes length
		return make([]byte, numBytes), len(data)
	}

	// Calculate the actual number of bytes to read, which may be less than numBytes
	// if the offset is near the end of data
	actualNumBytes := numBytes
	if offset+numBytes > len(data) {
		actualNumBytes = len(data) - offset
	}

	// Create a new byte array and copy the data from the original slice
	byteArray := make([]byte, actualNumBytes)
	copy(byteArray, data[offset:offset+actualNumBytes])

	return byteArray, offset + actualNumBytes
}

func (b *Blob) write(buffer []byte, pointer int) int {
	copy(b[pointer:], buffer)
	return pointer + len(buffer)
}

// ToData decodes the blob into raw byte data. See FromData above for details on the encoding
// format.
func (b *Blob) ToData() (Data, error) {
	data := make(Data, BlobSize)
	firstField := b[:FieldSize]

	// check the version
	if firstField[1] != EncodingVersion {
		return nil, fmt.Errorf("invalid blob, expected version %d, got %d", EncodingVersion, firstField[0])
	}

	// decode the 3-byte length prefix into 4-byte integer
	var dataLen int32

	// Assuming b[2], b[3], and b[4] contain the encoded length in big-endian format
	dataLen = int32(b[2]) << 16 // Shift the most significant byte 16 bits to the left
	dataLen |= int32(b[3]) << 8 // Shift the next byte 8 bits to the left and OR it with the current length
	dataLen |= int32(b[4])      // OR the least significant byte with the current length

	if dataLen > (int32)(len(data)) {
		return nil, fmt.Errorf("invalid blob, length prefix out of range: %d", dataLen)
	}

	// copy the first 27 bytes of the first field element into the output
	copy(data[0:27], firstField[5:])

	encodedByte := make([]byte, 4)
	encodedByte[0] = firstField[0]
	// copy the remaining 31*3 bytes of the first field into the output
	for i := 1; i < 4; i++ {
		// check that the highest order bit of the first byte of each field element is not set
		if firstField[i*32]&(1<<7) != 0 {
			return nil, fmt.Errorf("invalid blob, field element %d has highest order bit set", i)
		}
		encodedByte[i] = firstField[i*32]
		copy(data[27+31*(i-1)+i:], b[i*32+1:i*32+32])
	}

	x := (encodedByte[0] & 0b0011_1111) | ((encodedByte[1] & 0b0011_0000) << 2)
	y := (encodedByte[1] & 0b0000_1111) | ((encodedByte[3] & 0b0000_1111) << 4)
	z := (encodedByte[2] & 0b0011_1111) | ((encodedByte[3] & 0b0011_0000) << 2)
	data[27] = x
	data[27+31+1] = y
	data[27+31*2+2] = z

	// for loop to decode 128 bytes of data at a time from the next 4 field elements
	for i := 1; i < 1024; i++ {
		encodedByte := make([]byte, 4)
		for j := 0; j < 4; j++ {
			// check that the highest order bit of the first byte of each field element is not set
			if b[i*FieldSize+j*32]&(1<<7) != 0 {
				return nil, fmt.Errorf("invalid blob, field element %d has highest order bit set", i)
			}
			// record the first byte of each field element
			encodedByte[j] = b[i*FieldSize+j*32]
			// -4 because of 1 byte of version and 3 bytes of length prefix
			copy(data[FieldCapacity*i+j*31-4+j:FieldCapacity*i+(j+1)*31-4+j], b[i*FieldSize+j*32+1:])
		}
		x := (encodedByte[0] & 0b0011_1111) | ((encodedByte[1] & 0b0011_0000) << 2)
		y := (encodedByte[1] & 0b0000_1111) | ((encodedByte[3] & 0b0000_1111) << 4)
		z := (encodedByte[2] & 0b0011_1111) | ((encodedByte[3] & 0b0011_0000) << 2)
		// copy the decoded data into the output
		data[FieldCapacity*i+27] = x
		data[FieldCapacity*i+27+31*1+1] = y
		data[FieldCapacity*i+27+31*2+2] = z
	}
	data = data[:dataLen]
	return data, nil
}

func (b *Blob) Clear() {
	for i := 0; i < BlobSize; i++ {
		b[i] = 0
	}
}

func clearBuffer(buffer []byte) {
	for i := 0; i < len(buffer); i++ {
		buffer[i] = 0
	}
}
