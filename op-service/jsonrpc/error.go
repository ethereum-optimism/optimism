// Package jsonrpc defines JSON-RPC protocol types shared by RPC clients, servers and metrics.
package jsonrpc

import "fmt"

// Error is a JSON-RPC error object. Returned from a served RPC method, it is sent to the
// client with its Code and Data, and Error() as the message, as it implements the go-ethereum
// rpc.Error and rpc.DataError interfaces.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("json-rpc error %d", e.Code)
	}
	return e.Message
}

func (e *Error) ErrorCode() int {
	return e.Code
}

func (e *Error) ErrorData() any {
	return e.Data
}
