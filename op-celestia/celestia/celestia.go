package celestia

import (
	"encoding"
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidSize = errors.New("invalid size")
)

// Framer defines a way to encode/decode a FrameRef.
type Framer interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// FrameRef contains the reference to the specific frame on celestia and
// satisfies the Framer interface.
type FrameRef struct {
	BlockHeight  uint64
	TxCommitment []byte
}

var _ Framer = &FrameRef{}

// MarshalBinary encodes the FrameRef to binary
// serialization format: height + commitment
//
//	----------------------------------------
//
// | 8 byte uint64  |  32 byte commitment   |
//
//	----------------------------------------
//
// | <-- height --> | <-- commitment -->    |
//
//	----------------------------------------
func (f *FrameRef) MarshalBinary() ([]byte, error) {
	ref := make([]byte, 8+len(f.TxCommitment))

	binary.LittleEndian.PutUint64(ref, f.BlockHeight)
	copy(ref[8:], f.TxCommitment)

	return ref, nil
}

// UnmarshalBinary decodes the binary to FrameRef
// serialization format: height + commitment
//
//	----------------------------------------
//
// | 8 byte uint64  |  32 byte commitment   |
//
//	----------------------------------------
//
// | <-- height --> | <-- commitment -->    |
//
//	----------------------------------------
func (f *FrameRef) UnmarshalBinary(ref []byte) error {
	if len(ref) <= 8 {
		return ErrInvalidSize
	}
	f.BlockHeight = binary.LittleEndian.Uint64(ref[:8])
	f.TxCommitment = ref[8:]
	return nil
}
