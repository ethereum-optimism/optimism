package kvstore

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// ErrNotFound is returned when a pre-image cannot be found in the KV store.
var ErrNotFound = errors.New("not found")

// KV is a Key-Value store interface for pre-image data.
type KV interface {
	// Put puts the pre-image value v in the key-value store with key k.
	// KV store implementations may return additional errors specific to the KV storage.
	Put(k common.Hash, v []byte) error

	// Get retrieves the pre-image with key k from the key-value store.
	// It returns ErrNotFound when the pre-image cannot be found.
	// KV store implementations may return additional errors specific to the KV storage.
	Get(k common.Hash) ([]byte, error)

	// Closes the KV store.
	Close() error
}
