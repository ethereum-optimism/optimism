package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const MaxBlocksInL1Range = uint64(100)

type Source struct {
	client     *ethclient.Client
	downloader *Downloader
}

func NewSource(client *ethclient.Client) Source {
	return Source{
		client:     client,
		downloader: NewDownloader(client),
	}
}

func (s Source) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return s.client.SubscribeNewHead(ctx, ch)
}

func (s Source) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return s.client.HeaderByHash(ctx, hash)
}

func (s Source) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return s.client.HeaderByNumber(ctx, number)
}

func (s Source) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return s.client.TransactionReceipt(ctx, txHash)
}

func (s Source) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return s.client.BlockByHash(ctx, hash)
}

func (s Source) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return s.client.BlockByNumber(ctx, number)
}

func (s Source) Fetch(ctx context.Context, id eth.BlockID) (*types.Block, []*types.Receipt, error) {
	return s.downloader.Fetch(ctx, id)
}

func (s Source) Close() {
	s.client.Close()
}

func (s Source) FetchL1Info(ctx context.Context, id eth.BlockID) (derive.L1Info, error) {
	return s.client.BlockByHash(ctx, id.Hash)
}

func (s Source) FetchReceipts(ctx context.Context, id eth.BlockID) ([]*types.Receipt, error) {
	_, receipts, err := s.Fetch(ctx, id)
	return receipts, err
}

func (s Source) FetchTransactions(ctx context.Context, window []eth.BlockID) ([]*types.Transaction, error) {
	var txns []*types.Transaction
	for _, id := range window {
		block, err := s.client.BlockByHash(ctx, id.Hash)
		if err != nil {
			return nil, err
		}
		txns = append(txns, block.Transactions()...)
	}
	return txns, nil

}
func (s Source) L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error) {
	return s.l1BlockRefByNumber(ctx, nil)
}

func (s Source) L1BlockRefByNumber(ctx context.Context, l1Num uint64) (eth.L1BlockRef, error) {
	return s.l1BlockRefByNumber(ctx, new(big.Int).SetUint64(l1Num))
}

// l1BlockRefByNumber wraps l1.HeaderByNumber to return an eth.L1BlockRef
// This is internal because the exposed L1BlockRefByNumber takes uint64 instead of big.Ints
func (s Source) l1BlockRefByNumber(ctx context.Context, number *big.Int) (eth.L1BlockRef, error) {
	header, err := s.client.HeaderByNumber(ctx, number)
	if err != nil {
		// w%: wrap the error, we still need to detect if a canonical block is not found, a.k.a. end of chain.
		return eth.L1BlockRef{}, fmt.Errorf("failed to determine block-hash of height %v, could not get header: %w", number, err)
	}
	l1Num := header.Number.Uint64()
	parentNum := l1Num
	if parentNum > 0 {
		parentNum -= 1
	}
	return eth.L1BlockRef{
		Self:   eth.BlockID{Hash: header.Hash(), Number: l1Num},
		Parent: eth.BlockID{Hash: header.ParentHash, Number: parentNum},
	}, nil
}

// L1Range returns a range of L1 block beginning just after `begin`.
func (s Source) L1Range(ctx context.Context, begin eth.BlockID) ([]eth.BlockID, error) {
	// Ensure that we start on the expected chain.
	if canonicalBegin, err := s.L1BlockRefByNumber(ctx, begin.Number); err != nil {
		return nil, fmt.Errorf("failed to fetch L1 block %v %v: %w", begin.Number, begin.Hash, err)
	} else {
		if canonicalBegin.Self != begin {
			return nil, fmt.Errorf("Re-org at begin block. Expected: %v. Actual: %v", begin, canonicalBegin.Self)
		}
	}

	l1head, err := s.L1HeadBlockRef(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch head L1 block: %w", err)
	}
	maxBlocks := MaxBlocksInL1Range
	// Cap maxBlocks if there are less than maxBlocks between `begin` and the head of the chain.
	if l1head.Self.Number-begin.Number <= maxBlocks {
		maxBlocks = l1head.Self.Number - begin.Number
	}

	if maxBlocks == 0 {
		return nil, nil
	}

	prevHash := begin.Hash
	var res []eth.BlockID
	// TODO: Walk backwards to be able to use block by hash
	for i := begin.Number + 1; i < begin.Number+maxBlocks+1; i++ {
		n, err := s.L1BlockRefByNumber(ctx, i)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch L1 block %v: %w", i, err)
		}
		// TODO(Joshua): Look into why this fails around the genesis block
		if n.Parent.Number != 0 && n.Parent.Hash != prevHash {
			return nil, errors.New("re-organization occurred while attempting to get l1 range")
		}
		prevHash = n.Self.Hash
		res = append(res, n.Self)
	}

	return res, nil
}
