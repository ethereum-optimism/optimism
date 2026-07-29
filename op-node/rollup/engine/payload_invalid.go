package engine

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadInvalidEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
	Err      error
}

func (ev PayloadInvalidEvent) String() string {
	return "payload-invalid"
}
