package rpc

import "github.com/ethereum/go-ethereum/rpc"

var (
	_ rpc.Error = new(JsonError)
	_ error     = new(JsonError)
)

type JsonError struct {
	Message string
	Code    int
}

func (j *JsonError) Error() string {
	return j.Message
}

func (j *JsonError) ErrorCode() int {
	return j.Code
}
