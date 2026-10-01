package jsonrpc

import (
	"context"
	"encoding/json"
)

// Message is a JSON-RPC request, as seen by a Recorder.
type Message struct {
	Method string
	Params json.RawMessage
}

// Response is the response to a JSON-RPC request. Error is nil on success.
type Response struct {
	Result json.RawMessage
	Error  *Error
}

// RecordDone is called with the response to a recorded request, once it arrives.
// It is not called when no response arrives, e.g. on a transport failure or timeout.
type RecordDone func(ctx context.Context, resp Response)

// Recorder observes the JSON-RPC requests a client sends. RecordOutgoing is called before a request
// is sent, and may return nil when the response is of no interest. Implementations must be safe
// for concurrent use.
type Recorder interface {
	RecordOutgoing(ctx context.Context, msg Message) RecordDone
}
