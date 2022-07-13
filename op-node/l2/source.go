package l2

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Source struct {
	rpc     client.RPC    // raw RPC client. Used for the consensus namespace
	client  client.Client // go-ethereum's wrapper around the rpc client for the eth namespace
	genesis *rollup.Genesis
	log     log.Logger
}

func NewSource(l2Node client.RPC, l2Client client.Client, genesis *rollup.Genesis, log log.Logger) (*Source, error) {
	return &Source{
		rpc:     l2Node,
		client:  l2Client,
		genesis: genesis,
		log:     log,
	}, nil
}

func (s *Source) Close() {
	s.rpc.Close()
}

func (s *Source) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	// TODO: we really do not need to parse every single tx and block detail, keeping transactions encoded is faster.
	block, err := s.client.BlockByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L2 block by hash: %v", err)
	}
	payload, err := eth.BlockAsPayload(block)
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 block as payload: %w", err)
	}
	return payload, nil
}

func (s *Source) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayload, error) {
	// TODO: we really do not need to parse every single tx and block detail, keeping transactions encoded is faster.
	block, err := s.client.BlockByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L2 block by number: %v", err)
	}
	payload, err := eth.BlockAsPayload(block)
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 block as payload: %w", err)
	}
	return payload, nil
}

// ForkchoiceUpdate updates the forkchoice on the execution client. If attributes is not nil, the engine client will also begin building a block
// based on attributes after the new head block and return the payload ID.
// May return an error in ForkChoiceResult, but the error is marshalled into the error return
func (s *Source) ForkchoiceUpdate(ctx context.Context, fc *eth.ForkchoiceState, attributes *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	e := s.log.New("state", fc, "attr", attributes)
	e.Trace("Sharing forkchoice-updated signal")
	fcCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.ForkchoiceUpdatedResult
	err := s.rpc.CallContext(fcCtx, &result, "engine_forkchoiceUpdatedV1", fc, attributes)
	if err == nil {
		e.Trace("Shared forkchoice-updated signal")
		if attributes != nil {
			e.Trace("Received payload id", "payloadId", result.PayloadID)
		}
		return &result, nil
	} else {
		e = e.New("err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			e.Warn("Unexpected error code in forkchoice-updated response", "code", code)
		} else {
			e.Error("Failed to share forkchoice-updated signal")
		}
		return nil, err
	}
}

// ExecutePayload executes a built block on the execution engine and returns an error if it was not successful.
func (s *Source) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e := s.log.New("block_hash", payload.BlockHash)
	e.Trace("sending payload for execution")

	execCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.PayloadStatusV1
	err := s.rpc.CallContext(execCtx, &result, "engine_newPayloadV1", payload)
	e.Trace("Received payload execution result", "status", result.Status, "latestValidHash", result.LatestValidHash, "message", result.ValidationError)
	if err != nil {
		e.Error("Payload execution failed", "err", err)
		return nil, fmt.Errorf("failed to execute payload: %v", err)
	}
	return &result, nil
}

// GetPayload gets the execution payload associated with the PayloadId
func (s *Source) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	e := s.log.New("payload_id", payloadId)
	e.Trace("getting payload")
	var result eth.ExecutionPayload
	err := s.rpc.CallContext(ctx, &result, "engine_getPayloadV1", payloadId)
	if err != nil {
		e = e.New("payload_id", payloadId, "err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			if code != eth.UnavailablePayload {
				e.Warn("unexpected error code in get-payload response", "code", code)
			} else {
				e.Warn("unavailable payload in get-payload request")
			}
		} else {
			e.Error("failed to get payload")
		}
		return nil, err
	}
	e.Trace("Received payload")
	return &result, nil
}

// L2BlockRefHead returns the canonical block and parent ids.
func (s *Source) L2BlockRefHead(ctx context.Context) (eth.L2BlockRef, error) {
	block, err := s.client.BlockByNumber(ctx, nil)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of head, could not get header: %w", err)
	}
	return blockToBlockRef(block, s.genesis)
}

// L2BlockRefByNumber returns the canonical block and parent ids.
func (s *Source) L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error) {
	block, err := s.client.BlockByNumber(ctx, l2Num)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of height %v, could not get header: %w", l2Num, err)
	}
	return blockToBlockRef(block, s.genesis)
}

// L2BlockRefByHash returns the block & parent ids based on the supplied hash. The returned BlockRef may not be in the canonical chain
func (s *Source) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	block, err := s.client.BlockByHash(ctx, l2Hash)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of height %v, could not get header: %w", l2Hash, err)
	}
	return blockToBlockRef(block, s.genesis)
}

// blockToBlockRef extracts the essential L2BlockRef information from a block,
// falling back to genesis information if necessary.
func blockToBlockRef(block *types.Block, genesis *rollup.Genesis) (eth.L2BlockRef, error) {
	var l1Origin eth.BlockID
	var sequenceNumber uint64
	if block.NumberU64() == genesis.L2.Number {
		if block.Hash() != genesis.L2.Hash {
			return eth.L2BlockRef{}, fmt.Errorf("expected L2 genesis hash to match L2 block at genesis block number %d: %s <> %s", genesis.L2.Number, block.Hash(), genesis.L2.Hash)
		}
		l1Origin = genesis.L1
		sequenceNumber = 0
	} else {
		txs := block.Transactions()
		if len(txs) == 0 {
			return eth.L2BlockRef{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", block.Hash())
		}
		tx := txs[0]
		if tx.Type() != types.DepositTxType {
			return eth.L2BlockRef{}, fmt.Errorf("first block tx has unexpected tx type: %d", tx.Type())
		}
		info, err := derive.L1InfoDepositTxData(tx.Data())
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %v", err)
		}
		l1Origin = eth.BlockID{Hash: info.BlockHash, Number: info.Number}
		sequenceNumber = info.SequenceNumber
	}
	return eth.L2BlockRef{
		Hash:           block.Hash(),
		Number:         block.NumberU64(),
		ParentHash:     block.ParentHash(),
		Time:           block.Time(),
		L1Origin:       l1Origin,
		SequenceNumber: sequenceNumber,
	}, nil
}

type ReadOnlySource struct {
	rpc     client.RPC    // raw RPC client. Used for methods that do not already have bindings
	client  client.Client // go-ethereum's wrapper around the rpc client for the eth namespace
	genesis *rollup.Genesis
	log     log.Logger
}

func NewReadOnlySource(l2Node client.RPC, l2Client client.Client, genesis *rollup.Genesis, log log.Logger) (*ReadOnlySource, error) {
	return &ReadOnlySource{
		rpc:     l2Node,
		client:  l2Client,
		genesis: genesis,
		log:     log,
	}, nil
}

// TODO: de-duplicate Source and ReadOnlySource.
// We should really have a L1-downloader like binding that is more configurable and has caching.

// L2BlockRefByNumber returns the canonical block and parent ids.
func (s *ReadOnlySource) L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error) {
	block, err := s.client.BlockByNumber(ctx, l2Num)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of height %v, could not get header: %w", l2Num, err)
	}
	return blockToBlockRef(block, s.genesis)
}

// L2BlockRefByHash returns the block & parent ids based on the supplied hash. The returned BlockRef may not be in the canonical chain
func (s *ReadOnlySource) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	block, err := s.client.BlockByHash(ctx, l2Hash)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of height %v, could not get header: %w", l2Hash, err)
	}
	return blockToBlockRef(block, s.genesis)
}

func (s *ReadOnlySource) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return s.client.BlockByNumber(ctx, number)
}

func (s *ReadOnlySource) GetBlockHeader(ctx context.Context, blockTag string) (*types.Header, error) {
	var head *types.Header
	err := s.rpc.CallContext(ctx, &head, "eth_getBlockByNumber", blockTag, false)
	return head, err
}

func (s *ReadOnlySource) GetProof(ctx context.Context, address common.Address, blockTag string) (*AccountResult, error) {
	var getProofResponse *AccountResult
	err := s.rpc.CallContext(ctx, &getProofResponse, "eth_getProof", address, []common.Hash{}, blockTag)
	if err == nil && getProofResponse == nil {
		err = ethereum.NotFound
	}
	return getProofResponse, err
}
