package proxyd

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type RewriteContext struct {
	Latest        hexutil.Uint64
	Safe          hexutil.Uint64
	Finalized     hexutil.Uint64
	MaxBlockRange uint64
}

type RewriteResult uint8

const (
	RewriteNone RewriteResult = iota
	RewriteOverrideError
	RewriteOverrideRequest
	RewriteOverrideResponse
)

var (
	ErrRewriteBlockOutOfRange = errors.New("block is out of range")
	ErrRewriteRangeTooLarge   = errors.New("block range is too large")
)

func RewriteTags(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	rw, err := RewriteResponse(rctx, req, res)
	if rw == RewriteOverrideResponse {
		return rw, err
	}
	return RewriteRequest(rctx, req, res)
}

func RewriteResponse(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	switch req.Method {
	case "eth_blockNumber":
		res.Result = rctx.Latest
		return RewriteOverrideResponse, nil
	}
	return RewriteNone, nil
}

func RewriteRequest(rctx RewriteContext, req *RPCReq, res *RPCRes) (RewriteResult, error) {
	switch req.Method {
	case "eth_getLogs", "eth_newFilter":
		return rewriteRange(rctx, req, res, 0)
	case "debug_getRawReceipts", "consensus_getReceipts":
		return rewriteParam(rctx, req, res, 0, true, false)
	case "eth_getBalance", "eth_getCode", "eth_getTransactionCount", "eth_call":
		return rewriteParam(rctx, req, res, 1, false, true)
	case "eth_getStorageAt", "eth_getProof":
		return rewriteParam(rctx, req, res, 2, false, true)
	case "eth_getBlockTransactionCountByNumber", "eth_getUncleCountByBlockNumber", "eth_getBlockByNumber", "eth_getTransactionByBlockNumberAndIndex", "eth_getUncleByBlockNumberAndIndex":
		return rewriteParam(rctx, req, res, 0, false, false)
	}
	return RewriteNone, nil
}

func rewriteParam(rctx RewriteContext, req *RPCReq, res *RPCRes, pos int, required bool, blockNrOrHash bool) (RewriteResult, error) {
	var p []interface{}
	err := json.Unmarshal(req.Params, &p)
	if err != nil {
		return RewriteOverrideError, err
	}

	if len(p) == pos && !required {
		p = append(p, "latest")
	} else if len(p) <= pos {
		return RewriteNone, nil
	}

	var val interface{}
	var rw bool
	if blockNrOrHash {
		bnh, err := remarshalBlockNumberOrHash(p[pos])
		if err != nil {
			s, ok := p[pos].(string)
			if ok {
				val, rw, err = rewriteTag(rctx, s)
				if err != nil {
					return RewriteOverrideError, err
				}
			} else {
				return RewriteOverrideError, errors.New("expected BlockNumberOrHash or string")
			}
		} else {
			val, rw, err = rewriteTagWithContext(rctx, bnh)
			if err != nil {
				return RewriteOverrideError, err
			}
		}
	} else {
		s, ok := p[pos].(string)
		if !ok {
			return RewriteOverrideError, errors.New("expected string")
		}

		val, rw, err = rewriteTagWithContext(rctx, s)
		if err != nil {
			return RewriteOverrideError, err
		}
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

	_, hasFrom := p[pos]["fromBlock"]
	_, hasTo := p[pos]["toBlock"]
	if hasFrom && !hasTo {
		p[pos]["toBlock"] = "latest"
	} else if hasTo && !hasFrom {
		p[pos]["fromBlock"] = "latest"
	}

	modifiedFrom, err := rewriteTagMapWithContext(rctx, p[pos], "fromBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	modifiedTo, err := rewriteTagMapWithContext(rctx, p[pos], "toBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	if rctx.MaxBlockRange > 0 && (hasFrom || hasTo) {
		from, err := blockNumber(p[pos], "fromBlock", uint64(rctx.Latest))
		if err != nil {
			return RewriteOverrideError, err
		}
		to, err := blockNumber(p[pos], "toBlock", uint64(rctx.Latest))
		if err != nil {
			return RewriteOverrideError, err
		}
		if to-from > rctx.MaxBlockRange {
			return RewriteOverrideError, ErrRewriteRangeTooLarge
		}
	}

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

func blockNumber(m map[string]interface{}, key string, latest uint64) (uint64, error) {
	current, ok := m[key].(string)
	if !ok {
		return 0, errors.New("expected string")
	}

	if current == "earliest" {
		return 0, nil
	}
	if current == "pending" {
		return latest + 1, nil
	}
	return hexutil.DecodeUint64(current)
}

func rewriteTagMapWithContext(rctx RewriteContext, m map[string]interface{}, key string) (bool, error) {
	if m[key] == nil || m[key] == "" {
		return false, nil
	}

	current, ok := m[key].(string)
	if !ok {
		return false, errors.New("expected string")
	}

	val, rw, err := rewriteTagWithContext(rctx, current)
	if err != nil {
		return false, err
	}
	if rw {
		m[key] = val
		return true, nil
	}

	return false, nil
