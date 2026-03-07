package state

import "errors"

// ErrNotFound is returned when a state key does not exist.
var ErrNotFound = errors.New("state not found")

// Store is the interface for persisting state blobs across pipeline runs.
// Each blob is identified by a string key (e.g. "flake-db", "correlation-matrix").
//
// Implementations:
//   - LocalStore: filesystem (dev/testing)
//   - CircleCIStore: fetches from previous pipeline artifacts, saves locally
//     for upload via store_artifacts
type Store interface {
	// Load retrieves a state blob by key. Returns ErrNotFound if the key doesn't exist.
	Load(key string) ([]byte, error)

	// Save persists a state blob by key.
	Save(key string, data []byte) error
}
