package bindings

import (
	"github.com/HashKeyChain/verse/op-service/eth"
)

type EventLogger struct {
	EmitLog func(topics []eth.Bytes32, data []byte) TypedCall[any] `sol:"emitLog"`
}
