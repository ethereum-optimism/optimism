package proxyd

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type RPCRes struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCErr         `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
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

func IsValidID(id json.RawMessage) bool {
	// handle the case where the ID is a string
	if strings.HasPrefix(string(id), "\"") && strings.HasSuffix(string(id), "\"") {
		return len(id) > 2
	}

	// technically allows a boolean/null ID, but so does Geth
	// https://github.com/ethereum/go-ethereum/blob/master/rpc/json.go#L72
	return len(id) > 0 && id[0] != '{' && id[0] != '['
}

func ParseRPCReq(body []byte) (*RPCReq, error) {
	req := new(RPCReq)
	if err := json.Unmarshal(body, req); err != nil {
		return nil, ErrParseErr
	}

	return req, nil
}

func ParseBatchRPCReq(body []byte) ([]json.RawMessage, error) {
	batch := make([]json.RawMessage, 0)
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, err
	}

	return batch, nil
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

func ValidateRPCReq(req *RPCReq) error {
	if req.JSONRPC != JSONRPCVersion {
		return ErrInvalidRequest("invalid JSON-RPC version")
	}

	if req.Method == "" {
		return ErrInvalidRequest("no method specified")
	}

	if !IsValidID(req.ID) {
		return ErrInvalidRequest("invalid ID")
	}

	return nil
}

func NewRPCErrorRes(id json.RawMessage, err error) *RPCRes {
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
