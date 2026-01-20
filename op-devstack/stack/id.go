package stack

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ChainIDProvider presents a type that provides a relevant ChainID.
type ChainIDProvider interface {
	ChainID() eth.ChainID
}

// KindProvider presents a type that provides a relevant ComponentKind. E.g. KindL2Batcher.
type KindProvider interface {
	Kind() ComponentKind
}

// Keyed presents a type that provides a relevant string key. E.g. a named superchain.
type Keyed interface {
	Key() string
}

const maxIDLength = 100

var errInvalidID = errors.New("invalid ID")
