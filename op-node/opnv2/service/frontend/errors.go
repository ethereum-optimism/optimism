package frontend

import "github.com/ethereum/go-ethereum/rpc"

// ErrNotImplemented represents a standard error: (EIP-1474) Method is not implemented
var ErrNotImplemented = &rpc.JsonError{
	Code:    -32004,
	Message: "Method not supported",
	Data:    nil,
}
