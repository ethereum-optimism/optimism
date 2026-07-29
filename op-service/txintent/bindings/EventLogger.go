package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type EventLogger struct {
	EmitLog func(topics []eth.Bytes32, data []byte) TypedCall[any] `sol:"emitLog"`
}
