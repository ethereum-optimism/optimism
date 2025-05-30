package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type EventLoggerFactory struct {
	BaseCallFactory
}

func NewEventLoggerCallFactory(opts ...CallFactoryOption) *EventLoggerFactory {
	return &EventLoggerFactory{BaseCallFactory: *NewBaseCallFactory(opts...)}
}

type EventLogger struct {
	EventLoggerFactory

	EmitLog func(topics []eth.Bytes32, data []byte) TypedCall[any] `sol:"emitLog"`
}

func NewEventLogger(f *EventLoggerFactory) *EventLogger {
	eventLogger := EventLogger{EventLoggerFactory: *f}
	InitImpl(&eventLogger)
	return &eventLogger
}
