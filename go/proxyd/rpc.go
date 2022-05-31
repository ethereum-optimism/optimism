package proxyd

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type RPCRes struct {
	JSONRPC string
	Result  interface{}
	Error   *RPCErr
	ID      json.RawMessage
}

type rpcResJSON struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCErr         `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

type nullResultRPCRes struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result"`
	ID      json.RawMessage `json:"id"`
}

func (r *RPCRes) IsError() bool {
	return r.Error != nil
}

func (r *RPCRes) MarshalJSON() ([]byte, error) {
	if r.Result == nil && r.Error == nil {
		return json.Marshal(&nullResultRPCRes{
			JSONRPC: r.JSONRPC,
			Result:  nil,
			ID:      r.ID,
		})
	}

	return json.Marshal(&rpcResJSON{
		JSONRPC: r.JSONRPC,
		Result:  r.Result,
		Error:   r.Error,
		ID:      r.ID,
	})
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

func ValidateRPCReq(req *RPCReq, getBlockNum GetLatestBlockNumFn) error {
	if req.JSONRPC != JSONRPCVersion {
		return ErrInvalidRequest("invalid JSON-RPC version")
	}

	if req.Method == "" {
		return ErrInvalidRequest("no method specified")
	}

	if !IsValidID(req.ID) {
		return ErrInvalidRequest("invalid ID")
	}

	archive, err := isArchiveRequest(req, getBlockNum)
	if err != nil {
		// ignore errors and pass em up the request handler chain
		log.Warn("failed to decode request for archive", "err", err)
		return nil
	}
	if archive {
		return ErrInvalidRequest("unsupported archive request")
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

func IsBatch(raw []byte) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

func isArchiveRequest(req *RPCReq, getBlockNum GetLatestBlockNumFn) (bool, error) {
	const BLOCK_NUM_ARCHIVE_HEIGHT = 256

	var (
		tag            string
		latestBlockNum uint64
		err            error
	)
	switch req.Method {
	case "eth_call", "eth_getBalance", "eth_getTransactionCount":
		tag, err = extractTerminalBlockTag(req)
	case "eth_getBlockByNumber":
		tag, _, err = decodeGetBlockByNumberParams(req.Params)
	default:
		return false, nil
	}

	if err != nil {
		return false, err
	}
	if tag == "" || isBlockDependentParam(tag) {
		return false, nil
	}
	// NOTE: we're reading from an in-memory getBlockNum LVC, which doesn't block
	if latestBlockNum, err = getBlockNum(context.Background()); err != nil {
		return false, err
	}

	blockNum, err := decodeBlockInput(tag)
	if err != nil {
		return false, err
	}
	return blockNum+uint64(BLOCK_NUM_ARCHIVE_HEIGHT) < latestBlockNum, nil
}

// quick hack: if the request looks like a request containing a quantity|tag at the end, then return the tag
func extractTerminalBlockTag(req *RPCReq) (string, error) {
	var input []json.RawMessage
	if err := json.Unmarshal(req.Params, &input); err != nil {
		return "", err
	}
	if len(input) < 2 {
		return "", nil
	}
	var blockTag string
	err := json.Unmarshal(input[1], &blockTag)
	return blockTag, err
}
