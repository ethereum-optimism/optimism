package jsonrpc

import (
	"context"
	"encoding/json"
)

// Message is a JSON-RPC request or notification, as seen by a Recorder.
type Message struct {
	Method string
	// Params holds the encoded params of a request. It may be left empty for a notification.
	Params json.RawMessage
	// Notification is set for messages without an ID, which get no response.
	Notification bool
}

// Response is the response to a JSON-RPC request. Error is nil on success.
type Response struct {
	Result json.RawMessage
	Error  *Error
}

// RecordDone is called with the response to a recorded request, once it arrives.
// It is not called when no response arrives, e.g. on a transport failure or timeout.
type RecordDone func(ctx context.Context, resp Response)

// Recorder observes JSON-RPC traffic: RecordIncoming for received messages, RecordOutgoing for
// sent ones. Both are called before the message is handled or sent, and may return nil when the
// response is of no interest. The returned RecordDone is never called for a notification.
// Implementations must be safe for concurrent use.
type Recorder interface {
	RecordIncoming(ctx context.Context, msg Message) RecordDone
	RecordOutgoing(ctx context.Context, msg Message) RecordDone
}
