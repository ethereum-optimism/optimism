package engine

import "fmt"

// Kind identifies the engine client's kind, used to control the behavior of optimism in different engine clients.
type Kind string

const (
	Geth   Kind = "geth"
	Reth   Kind = "reth"
	Erigon Kind = "erigon"
)

var Kinds = []Kind{
	Geth,
	Reth,
	Erigon,
}

func (kind Kind) String() string {
	return string(kind)
}

func (kind *Kind) Set(value string) error {
	if !ValidEngineKind(Kind(value)) {
		return fmt.Errorf("unknown engine client kind: %q", value)
	}
	*kind = Kind(value)
	return nil
}

func (kind *Kind) Clone() any {
	cpy := *kind
	return &cpy
}

func (kind Kind) SupportsPostFinalizationELSync() bool {
	switch kind {
	case Geth:
		return false
	case Erigon, Reth:
		return true
	}
	return false
}

func ValidEngineKind(value Kind) bool {
	for _, k := range Kinds {
		if k == value {
			return true
		}
	}
	return false
}
