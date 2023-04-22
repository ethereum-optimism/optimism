package preimage

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

type Key interface {
	// PreimageKey changes the Key commitment into a
	// 32-byte type-prefixed preimage key.
	PreimageKey() common.Hash
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
	// Keccak25Key6Type is for keccak256 pre-images, for any global shared pre-images.
	Keccak25Key6Type KeyType = 2
)

// LocalIndexKey is a key local to the program, indexing a special program input.
type LocalIndexKey uint64

func (k LocalIndexKey) PreimageKey() (out common.Hash) {
	out[0] = byte(LocalKeyType)
	binary.BigEndian.PutUint64(out[24:], uint64(k))
	return
}

// Keccak256Key wraps a keccak256 hash to use it as a typed pre-image key.
type Keccak256Key common.Hash

func (k Keccak256Key) PreimageKey() (out common.Hash) {
	out = common.Hash(k)            // copy the keccak hash
	out[0] = byte(Keccak25Key6Type) // apply prefix
	return
}

func (k Keccak256Key) String() string {
	return common.Hash(k).String()
}

func (k Keccak256Key) TerminalString() string {
	return common.Hash(k).String()
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
