package engine

import "fmt"

// EngineClientKind identifies the engine client's kind, used to control the behavior of optimism in different engine clients.
type EngineClientKind string

const (
	EngineClientGeth   EngineClientKind = "geth"
	EngineClientReth   EngineClientKind = "reth"
	EngineClientErigon EngineClientKind = "erigon"
)

var EngineClientKinds = []EngineClientKind{
	EngineClientGeth,
	EngineClientReth,
	EngineClientErigon,
}

func (kind EngineClientKind) String() string {
	return string(kind)
}

func (kind *EngineClientKind) Set(value string) error {
	if !ValidEngineClientKind(EngineClientKind(value)) {
		return fmt.Errorf("unknown engine client kind: %q", value)
	}
	*kind = EngineClientKind(value)
	return nil
}

func (kind *EngineClientKind) Clone() any {
	cpy := *kind
	return &cpy
}

func ValidEngineClientKind(value EngineClientKind) bool {
	for _, k := range EngineClientKinds {
		if k == value {
			return true
		}
	}
	return false
}
