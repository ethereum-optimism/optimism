package l2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
	cll2 "github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// Require the fetching oracle to implement StateOracle
var _ cll2.StateOracle = (*FetchingL2Oracle)(nil)

const headBlockNumber = 1000

func TestNodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	hash := testutils.RandomHash(rng)

	t.Run("Error", func(t *testing.T) {
		stub := &stubCallContext{
			nextErr: errors.New("oops"),
		}
		fetcher := newFetcher(nil, stub)

		require.Panics(t, func() {
			fetcher.NodeByHash(hash)
		})
	})

	t.Run("Success", func(t *testing.T) {
		expected := (hexutil.Bytes)([]byte{12, 34})
		stub := &stubCallContext{
			nextResult: expected,
		}
		fetcher := newFetcher(nil, stub)

		node := fetcher.NodeByHash(hash)
		require.EqualValues(t, expected, node)
	})

	t.Run("RequestArgs", func(t *testing.T) {
		stub := &stubCallContext{
			nextResult: (hexutil.Bytes)([]byte{12, 34}),
		}
		fetcher := newFetcher(nil, stub)

		fetcher.NodeByHash(hash)
		require.Len(t, stub.requests, 1, "should make single request")
		req := stub.requests[0]
		require.Equal(t, "debug_dbGet", req.method)
		require.Equal(t, []interface{}{hash.Hex()}, req.args)
	})
}

func TestCodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	hash := testutils.RandomHash(rng)

	t.Run("Error", func(t *testing.T) {
		stub := &stubCallContext{
			nextErr: errors.New("oops"),
		}
		fetcher := newFetcher(nil, stub)

		require.Panics(t, func() { fetcher.CodeByHash(hash) })
	})

	t.Run("Success", func(t *testing.T) {
		expected := (hexutil.Bytes)([]byte{12, 34})
		stub := &stubCallContext{
			nextResult: expected,
		}
		fetcher := newFetcher(nil, stub)

		node := fetcher.CodeByHash(hash)
		require.EqualValues(t, expected, node)
	})

	t.Run("RequestArgs", func(t *testing.T) {
		stub := &stubCallContext{
			nextResult: (hexutil.Bytes)([]byte{12, 34}),
		}
		fetcher := newFetcher(nil, stub)

		fetcher.CodeByHash(hash)
		require.Len(t, stub.requests, 1, "should make single request")
		req := stub.requests[0]
		require.Equal(t, "debug_dbGet", req.method)
		codeDbKey := append(rawdb.CodePrefix, hash.Bytes()...)
		require.Equal(t, []interface{}{hexutil.Encode(codeDbKey)}, req.args)
	})

	t.Run("FallbackToUnprefixed", func(t *testing.T) {
		stub := &stubCallContext{
			nextErr: errors.New("not found"),
		}
		fetcher := newFetcher(nil, stub)

		// Panics because the code can't be found with or without the prefix
		require.Panics(t, func() { fetcher.CodeByHash(hash) })
		require.Len(t, stub.requests, 2, "should request with and without prefix")
		req := stub.requests[0]
		require.Equal(t, "debug_dbGet", req.method)
		codeDbKey := append(rawdb.CodePrefix, hash.Bytes()...)
		require.Equal(t, []interface{}{hexutil.Encode(codeDbKey)}, req.args)

		req = stub.requests[1]
		require.Equal(t, "debug_dbGet", req.method)
		codeDbKey = hash.Bytes()
		require.Equal(t, []interface{}{hexutil.Encode(codeDbKey)}, req.args)
	})
}

func TestBlockByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	hash := testutils.RandomHash(rng)

	t.Run("Success", func(t *testing.T) {
		block := blockWithNumber(rng, headBlockNumber-1)
		stub := &stubBlockSource{nextResult: block}
		fetcher := newFetcher(stub, nil)

		res := fetcher.BlockByHash(hash)
		require.Same(t, block, res)
	})

	t.Run("Error", func(t *testing.T) {
		stub := &stubBlockSource{nextErr: errors.New("boom")}
		fetcher := newFetcher(stub, nil)

		require.Panics(t, func() {
			fetcher.BlockByHash(hash)
		})
	})

	t.Run("RequestArgs", func(t *testing.T) {
		stub := &stubBlockSource{nextResult: blockWithNumber(rng, 1)}
		fetcher := newFetcher(stub, nil)

		fetcher.BlockByHash(hash)

		require.Len(t, stub.requests, 1, "should make single request")
		req := stub.requests[0]
		require.Equal(t, hash, req.blockHash)
	})

	t.Run("PanicWhenBlockAboveHeadRequested", func(t *testing.T) {
		// Block that the source can provide but is above the head block number
		block := blockWithNumber(rng, headBlockNumber+1)

		stub := &stubBlockSource{nextResult: block}
		fetcher := newFetcher(stub, nil)

		require.Panics(t, func() {
			fetcher.BlockByHash(block.Hash())
		})
	})
}

func blockWithNumber(rng *rand.Rand, num int64) *types.Block {
	header := testutils.RandomHeader(rng)
	header.Number = big.NewInt(num)
	return types.NewBlock(header, nil, nil, nil, trie.NewStackTrie(nil))
}

type blockRequest struct {
	ctx       context.Context
	blockHash common.Hash
}

type stubBlockSource struct {
	requests   []blockRequest
	nextErr    error
	nextResult *types.Block
}

func (s *stubBlockSource) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	s.requests = append(s.requests, blockRequest{
		ctx:       ctx,
		blockHash: blockHash,
	})
	return s.nextResult, s.nextErr
}

type callContextRequest struct {
	ctx    context.Context
	method string
	args   []interface{}
}

type stubCallContext struct {
	nextResult any
	nextErr    error
	requests   []callContextRequest
}

func (c *stubCallContext) CallContext(ctx context.Context, result any, method string, args ...interface{}) error {
	if result != nil && reflect.TypeOf(result).Kind() != reflect.Ptr {
		return fmt.Errorf("call result parameter must be pointer or nil interface: %v", result)
	}
	c.requests = append(c.requests, callContextRequest{ctx: ctx, method: method, args: args})
	if c.nextErr != nil {
		return c.nextErr
	}
	res, err := json.Marshal(c.nextResult)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	err = json.Unmarshal(res, result)
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

func newFetcher(blockSource BlockSource, callContext CallContext) *FetchingL2Oracle {
	rng := rand.New(rand.NewSource(int64(1)))
	head := testutils.MakeBlockInfo(func(i *testutils.MockBlockInfo) {
		i.InfoNum = headBlockNumber
	})(rng)

	return &FetchingL2Oracle{
		ctx:         context.Background(),
		logger:      log.New(),
		head:        head,
		blockSource: blockSource,
		callContext: callContext,
	}
}
