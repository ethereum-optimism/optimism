package proxyd

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type RewriteContext struct {
	latest hexutil.Uint64
}

type RewriteResult uint8

const (
	// RewriteNone means request should be forwarded as-is
	RewriteNone RewriteResult = iota

	// RewriteOverrideError means there was an error attempting to rewrite
	RewriteOverrideError

	// RewriteOverrideRequest means the modified request should be forwarded to the backend
	RewriteOverrideRequest

	// RewriteOverrideResponse means to skip calling the backend and serve the overridden response
	RewriteOverrideResponse
)

// RewriteResponse modifies the response object to comply with the rewrite context
// after the method has been called at the backend
// RewriteResult informs the decision of the rewrite
func RewriteResponse(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	switch req.Method {
	case "eth_blockNumber":
		res.Result = rctx.latest
		return RewriteOverrideResponse, nil
	}
	return RewriteNone, nil
}

// RewriteRequest modifies the request object to comply with the rewrite context
// before the method has been called at the backend
// it returns false if nothing was changed
func RewriteRequest(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	switch req.Method {
	case "eth_getLogs",
		"eth_newFilter":
		return rewriteRange(rctx, req, res, 0)
	case "eth_getBalance",
		"eth_getCode",
		"eth_getTransactionCount",
		"eth_call":
		return rewriteParam(rctx, req, res, 1)
	case "eth_getStorageAt":
		return rewriteParam(rctx, req, res, 2)
	case "eth_getBlockTransactionCountByNumber",
		"eth_getUncleCountByBlockNumber",
		"eth_getBlockByNumber",
		"eth_getTransactionByBlockNumberAndIndex",
		"eth_getUncleByBlockNumberAndIndex":
		return rewriteParam(rctx, req, res, 0)
	}
	return RewriteNone, nil
}

func rewriteParam(rctx RewriteContext, req *RPCReq, res *RPCRes, pos int) (RewriteResult, error) {
	var p []interface{}
	err := json.Unmarshal(req.Params, &p)
	if err != nil {
		return RewriteOverrideError, err
	}

	if len(p) <= pos {
		p = append(p, "latest")
	}

	val, rw, err := rewriteTag(rctx, p[pos].(string))
	if err != nil {
		return RewriteOverrideError, err
	}

	if rw {
		p[pos] = val
		paramRaw, err := json.Marshal(p)
		if err != nil {
			return RewriteOverrideError, err
		}
		req.Params = paramRaw
		return RewriteOverrideRequest, nil
	}
	return RewriteNone, nil
}

func rewriteTag(rctx RewriteContext, current string) (string, bool, error) {
	param := current
	rw := false

	if param == "latest" {
		param = rctx.latest.String()
		rw = true
	}

	if strings.HasPrefix(param, "0x") {
		decode, err := hexutil.DecodeUint64(param)
		if err != nil {
			return "", false, err
		}
		b := hexutil.Uint64(decode)
		if b > rctx.latest {
			param = rctx.latest.String()
			rw = true
		}
	}
	return param, rw, nil
}

func rewriteRange(rctx RewriteContext, req *RPCReq, res *RPCRes, pos int) (RewriteResult, error) {
	var p []map[string]interface{}
	err := json.Unmarshal(req.Params, &p)
	if err != nil {
		return RewriteOverrideError, err
	}

	rw := false

	r, err := rewriteTagMap(rctx, p[pos], "fromBlock")
	if err != nil {
		return RewriteOverrideError, err
	}
	rw = r || rw

	r, err = rewriteTagMap(rctx, p[pos], "toBlock")
	if err != nil {
		return RewriteOverrideError, err
	}
	rw = r || rw

	if rw {
		paramsRaw, err := json.Marshal(p)
		req.Params = paramsRaw
		if err != nil {
			return RewriteOverrideError, err
		}
		return RewriteOverrideRequest, nil
	}

	return RewriteNone, nil
}

func rewriteTagMap(rctx RewriteContext, m map[string]interface{}, key string) (bool, error) {
	if m[key] == nil || m[key] == "" {
		return false, nil
	}

	val, ok := m[key].(string)
	if !ok {
		return false, errors.New("expected string")
	}

	rw := false

	if val == "latest" {
		m[key] = rctx.latest.String()
		rw = true
	}

	if strings.HasPrefix(val, "0x") {
		decode, err := hexutil.DecodeUint64(val)
		if err != nil {
			return false, err
		}
		b := hexutil.Uint64(decode)
		if b > rctx.latest {
			m[key] = rctx.latest.String()
			rw = true
		}
	}

	return rw, nil
}
