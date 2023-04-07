package kvstore

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// NotFoundErr is returned when a pre-image cannot be found in the KV store.
var NotFoundErr = errors.New("not found")

// KV is a Key-Value store interface for pre-image data.
type KV interface {
	Put(k common.Hash, v []byte) error
	Get(k common.Hash) ([]byte, error)
}
