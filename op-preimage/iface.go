package preimage

import (
	"encoding/binary"
	"encoding/hex"
)

type Key interface {
	// PreimageKey changes the Key commitment into a
	// 32-byte type-prefixed preimage key.
	PreimageKey() [32]byte
}

type Oracle interface {
	// Get the full pre-image of a given pre-image key.
	// This returns no error: the client state-transition
	// is invalid if there is any missing pre-image data.
	Get(key Key) []byte
}

type OracleFn func(key Key) []byte

func (fn OracleFn) Get(key Key) []byte {
	return fn(key)
}

// KeyType is the key-type of a pre-image, used to prefix the pre-image key with.
type KeyType byte

const (
	// The zero key type is illegal to use, ensuring all keys are non-zero.
	_ KeyType = 0
	// LocalKeyType is for input-type pre-images, specific to the local program instance.
	LocalKeyType KeyType = 1
	// Keccak256KeyType is for keccak256 pre-images, for any global shared pre-images.
	Keccak256KeyType KeyType = 2
	// GlobalGenericKeyType is a reserved key type for generic global data.
	GlobalGenericKeyType KeyType = 3
	// Sha256KeyType is for sha256 pre-images, for any global shared pre-images.
	Sha256KeyType KeyType = 4
	// BlobKeyType is for blob point pre-images.
	BlobKeyType KeyType = 5
	// PrecompileKeyType is for precompile result pre-images.
	PrecompileKeyType KeyType = 6
)

// LocalIndexKey is a key local to the program, indexing a special program input.
type LocalIndexKey uint64

func (k LocalIndexKey) PreimageKey() (out [32]byte) {
	out[0] = byte(LocalKeyType)
	binary.BigEndian.PutUint64(out[24:], uint64(k))
	return
}

// Keccak256Key wraps a keccak256 hash to use it as a typed pre-image key.
type Keccak256Key [32]byte

func (k Keccak256Key) PreimageKey() (out [32]byte) {
	out = k                         // copy the keccak hash
	out[0] = byte(Keccak256KeyType) // apply prefix
	return
}

func (k Keccak256Key) String() string {
	return "0x" + hex.EncodeToString(k[:])
}

func (k Keccak256Key) TerminalString() string {
	return "0x" + hex.EncodeToString(k[:])
}

// Sha256Key wraps a sha256 hash to use it as a typed pre-image key.hash
type Sha256Key [32]byte

func (k Sha256Key) PreimageKey() (out [32]byte) {
	out = k
	out[0] = byte(Sha256KeyType)
	return
}

func (k Sha256Key) String() string {
	return "0x" + hex.EncodeToString(k[:])
}

func (k Sha256Key) TerminalString() string {
	return "0x" + hex.EncodeToString(k[:])
}

// BlobKey is the hash of a blob commitment and `z` value to use as a preimage key for `y`.
type BlobKey [32]byte

func (k BlobKey) PreimageKey() (out [32]byte) {
	out = k
	out[0] = byte(BlobKeyType)
	return
}

func (k BlobKey) String() string {
	return "0x" + hex.EncodeToString(k[:])
}

func (k BlobKey) TerminalString() string {
	return "0x" + hex.EncodeToString(k[:])
}

// PrecompileKey is the hash of precompile address and its input data
type PrecompileKey [32]byte

func (k PrecompileKey) PreimageKey() (out [32]byte) {
	out = k
	out[0] = byte(PrecompileKeyType)
	return
}

func (k PrecompileKey) String() string {
	return "0x" + hex.EncodeToString(k[:])
}

func (k PrecompileKey) TerminalString() string {
	return "0x" + hex.EncodeToString(k[:])
}

// Hint is an interface to enable any program type to function as a hint,
// when passed to the Hinter interface, returning a string representation
// of what data the host should prepare pre-images for.
type Hint interface {
	Hint() string
}

// Hinter is an interface to write hints to the host.
// This may be implemented as a no-op or logging hinter
// if the program is executing in a read-only environment
// where the host is expected to have all pre-images ready.
type Hinter interface {
	Hint(v Hint)
}

type HinterFn func(v Hint)

func (fn HinterFn) Hint(v Hint) {
	fn(v)
}
