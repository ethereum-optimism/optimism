package proxyd

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/redis/go-redis/v9"
)

type RewriteContext struct {
	latest               hexutil.Uint64
	safe                 hexutil.Uint64
	finalized            hexutil.Uint64
	maxBlockRange        uint64
	cp                   *ConsensusPoller
	consensusPollerRetry bool
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
	ErrRewriteRangeTooLarge   = errors.New("block range is too large")
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
	case "debug_getRawReceipts", "consensus_getReceipts":
		return rewriteParam(rctx, req, res, 0, true, false)
	case "eth_getBalance",
		"eth_getCode",
		"eth_getTransactionCount",
		"eth_call":
		return rewriteParam(rctx, req, res, 1, false, true)
	case "eth_getStorageAt",
		"eth_getProof":
		return rewriteParam(rctx, req, res, 2, false, true)
	case "eth_getBlockTransactionCountByNumber",
		"eth_getUncleCountByBlockNumber",
		"eth_getBlockByNumber",
		"eth_getTransactionByBlockNumberAndIndex",
		"eth_getUncleByBlockNumberAndIndex":
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

	// we assume latest if the param is missing,
	// and we don't rewrite if there is not enough params
	if len(p) == pos && !required {
		p = append(p, "latest")
	} else if len(p) <= pos {
		return RewriteNone, nil
	}

	// support for https://eips.ethereum.org/EIPS/eip-1898
	var val interface{}
	var rw bool
	if blockNrOrHash {
		bnh, err := remarshalBlockNumberOrHash(p[pos])
		if err != nil {
			// fallback to string
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
			val, rw, err = rewriteTagBlockNumberOrHash(rctx, bnh)
			if err != nil {
				return RewriteOverrideError, err
			}
		}
	} else {
		s, ok := p[pos].(string)
		if !ok {
			return RewriteOverrideError, errors.New("expected string")
		}

		val, rw, err = rewriteTag(rctx, s)
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

	// if either fromBlock or toBlock is defined, default the other to "latest" if unset
	_, hasFrom := p[pos]["fromBlock"]
	_, hasTo := p[pos]["toBlock"]
	if hasFrom && !hasTo {
		p[pos]["toBlock"] = "latest"
	} else if hasTo && !hasFrom {
		p[pos]["fromBlock"] = "latest"
	}

	modifiedFrom, err := rewriteTagMap(rctx, p[pos], "fromBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	modifiedTo, err := rewriteTagMap(rctx, p[pos], "toBlock")
	if err != nil {
		return RewriteOverrideError, err
	}

	if rctx.maxBlockRange > 0 && (hasFrom || hasTo) {
		from, err := blockNumber(p[pos], "fromBlock", uint64(rctx.latest))
		if err != nil {
			return RewriteOverrideError, err
		}
		to, err := blockNumber(p[pos], "toBlock", uint64(rctx.latest))
		if err != nil {
			return RewriteOverrideError, err
		}
		if to-from > rctx.maxBlockRange {
			return RewriteOverrideError, ErrRewriteRangeTooLarge
		}
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

func blockNumber(m map[string]interface{}, key string, latest uint64) (uint64, error) {
	current, ok := m[key].(string)
	if !ok {
		return 0, errors.New("expected string")
	}
	// the latest/safe/finalized tags are already replaced by rewriteTag
	if current == "earliest" {
		return 0, nil
	}
	if current == "pending" {
		return latest + 1, nil
	}
	return hexutil.DecodeUint64(current)
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

func remarshalBlockNumberOrHash(current interface{}) (*rpc.BlockNumberOrHash, error) {
	jv, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}

	var bnh rpc.BlockNumberOrHash
	err = bnh.UnmarshalJSON(jv)
	if err != nil {
		return nil, err
	}

	return &bnh, nil
}

func rewriteTag(rctx RewriteContext, current string) (string, bool, error) {
	bnh, err := remarshalBlockNumberOrHash(current)
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
		err := tryConsensusBlockUpdate(bnh.BlockNumber.Int64(), int64(rctx.latest), rctx.cp, rctx.consensusPollerRetry)
		if err != nil {
			return "", false, err
		}
	}

	return current, false, nil
}

func rewriteTagBlockNumberOrHash(rctx RewriteContext, current *rpc.BlockNumberOrHash) (*rpc.BlockNumberOrHash, bool, error) {
	// this is a hash, not a block number
	if current.BlockNumber == nil {
		return current, false, nil
	}

	switch *current.BlockNumber {
	case rpc.PendingBlockNumber,
		rpc.EarliestBlockNumber:
		return current, false, nil
	case rpc.FinalizedBlockNumber:
		bn := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(rctx.finalized))
		return &bn, true, nil
	case rpc.SafeBlockNumber:
		bn := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(rctx.safe))
		return &bn, true, nil
	case rpc.LatestBlockNumber:
		bn := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(rctx.latest))
		return &bn, true, nil
	default:
		err := tryConsensusBlockUpdate(current.BlockNumber.Int64(), int64(rctx.latest), rctx.cp, rctx.consensusPollerRetry)
		if err != nil {
			return nil, false, err
		}
	}

	return current, false, nil
}

func tryConsensusBlockUpdate(requestedBlock int64, consensusBlock int64, cp *ConsensusPoller, consensusPollerRetry bool) error {
	totalSleepMs := 0
	retries := 0
	if requestedBlock > consensusBlock {
		if !consensusPollerRetry {
			return ErrRewriteBlockOutOfRange
		}

		// consensusHA mode
		// try to re-read the current state from redis and update in-memory
		if cp.consensusHA {
			ct := cp.tracker.(*RedisConsensusTracker)
			// nothing to do if this is the leader
			if ct.leader {
				return ErrRewriteBlockOutOfRange
			}

			key := ct.key("mutex")
			val, err := ct.client.Get(ct.ctx, key).Result()
			if err != nil && err != redis.Nil {
				log.Error("failed to read the lock", "err", err)
				RecordGroupConsensusError(ct.backendGroup, "read_lock", err)
			}
			ct.updateRemote(val)
		}

		// check if consensus has a newer block already
		// increase the sleep time in each iteration by 10ms
		// this will sleep at most 2100ms in total (20*21/2*10)
		for retries < 21 {
			retries += 1
			consensusBlock = int64(cp.GetLatestBlockNumber())
			if requestedBlock <= consensusBlock {
				break
			}
			sleepIterationMs := retries * 10
			totalSleepMs += sleepIterationMs
			time.Sleep(time.Duration(sleepIterationMs) * time.Millisecond)
		}
	}

	// track requested values
	RecordConsensusBlockUpdateRetries("rewriteTag", hexutil.Uint64(retries))
	RecordConsensusBlockUpdateTotalSleepMs("rewriteTag", hexutil.Uint64(totalSleepMs))
	RecordConsensusRequestedBlock("rewriteTag", hexutil.Uint64(requestedBlock))
	RecordConsensusCurrentConsensusBlock("rewriteTag", hexutil.Uint64(consensusBlock))

	// return an error if the consensus block is still too small
	if requestedBlock > consensusBlock {
		return ErrRewriteBlockOutOfRange
	}
	return nil
}
