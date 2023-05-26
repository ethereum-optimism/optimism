package proxyd

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type RewriteContext struct {
	latest    hexutil.Uint64
	safe      hexutil.Uint64
	finalized hexutil.Uint64
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

var (
	ErrRewriteBlockOutOfRange = errors.New("block is out of range")
)

// RewriteTags modifies the request and the response based on block tags
func RewriteTags(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	rw, err := RewriteResponse(rctx, req, res)
	if rw == RewriteOverrideResponse {
		return rw, err
	}
	return RewriteRequest(rctx, req, res)
}

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

	// we assume latest if the param is missing,
	// and we don't rewrite if there is not enough params
	if len(p) == pos {
		p = append(p, "latest")
	} else if len(p) < pos {
		return RewriteNone, nil
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

func rewriteRange(rctx RewriteContext, req *RPCReq, res *RPCRes, pos int) (RewriteResult, error) {
	var p []map[string]interface{}
	err := json.Unmarshal(req.Params, &p)
	if err != nil {
		return RewriteOverrideError, err
	}

	modifiedFrom, err := rewriteTagMap(rctx, p[pos], "fromBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	modifiedTo, err := rewriteTagMap(rctx, p[pos], "toBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	// if any of the fields the request have been changed, re-marshal the params
	if modifiedFrom || modifiedTo {
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

	current, ok := m[key].(string)
	if !ok {
		return false, errors.New("expected string")
	}

	val, rw, err := rewriteTag(rctx, current)
	if err != nil {
		return false, err
	}
	if rw {
		m[key] = val
		return true, nil
	}

	return false, nil
}

func rewriteTag(rctx RewriteContext, current string) (string, bool, error) {
	jv, err := json.Marshal(current)
	if err != nil {
		return "", false, err
	}

	var bnh rpc.BlockNumberOrHash
	err = bnh.UnmarshalJSON(jv)
	if err != nil {
		return "", false, err
	}

	// this is a hash, not a block
	if bnh.BlockNumber == nil {
		return current, false, nil
	}

	switch *bnh.BlockNumber {
	case rpc.PendingBlockNumber,
		rpc.EarliestBlockNumber:
		return current, false, nil
	case rpc.FinalizedBlockNumber:
		return rctx.finalized.String(), true, nil
	case rpc.SafeBlockNumber:
		return rctx.safe.String(), true, nil
	case rpc.LatestBlockNumber:
		return rctx.latest.String(), true, nil
	default:
		if bnh.BlockNumber.Int64() > int64(rctx.latest) {
			return "", false, ErrRewriteBlockOutOfRange
		}
	}

	return current, false, nil
}
