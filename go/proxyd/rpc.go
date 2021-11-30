package proxyd

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      *int            `json:"id"`
}

type RPCRes struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCErr     `json:"error,omitempty"`
	ID      *int        `json:"id"`
}

func (r *RPCRes) IsError() bool {
	return r.Error != nil
}

type RPCErr struct {
	Code          int    `json:"code"`
	Message       string `json:"message"`
	HTTPErrorCode int    `json:"-"`
}

func (r *RPCErr) Error() string {
	return r.Message
}

func ParseRPCReq(r io.Reader) (*RPCReq, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, wrapErr(err, "error reading request body")
	}

	req := new(RPCReq)
	if err := json.Unmarshal(body, req); err != nil {
		return nil, ErrParseErr
	}

	if req.JSONRPC != JSONRPCVersion {
		return nil, ErrInvalidRequest
	}

	if req.Method == "" {
		return nil, ErrInvalidRequest
	}

	return req, nil
}

func ParseRPCRes(r io.Reader) (*RPCRes, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, wrapErr(err, "error reading RPC response")
	}

	res := new(RPCRes)
	if err := json.Unmarshal(body, res); err != nil {
		return nil, wrapErr(err, "error unmarshaling RPC response")
	}

	return res, nil
}

func NewRPCErrorRes(id *int, err error) *RPCRes {
	var rpcErr *RPCErr
	if rr, ok := err.(*RPCErr); ok {
		rpcErr = rr
	} else {
		rpcErr = &RPCErr{
			Code:    JSONRPCErrorInternal,
			Message: err.Error(),
		}
	}

	return &RPCRes{
		JSONRPC: JSONRPCVersion,
		Error:   rpcErr,
		ID:      id,
	}
}
