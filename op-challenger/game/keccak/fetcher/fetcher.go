package fetcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNoLeavesFound = errors.New("no leaves found in block")
)

type L1Source interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

type Oracle interface {
	Addr() common.Address
	GetLeafBlocks(ctx context.Context, block batching.Block, ident gameTypes.LargePreimageIdent) ([]uint64, error)
	DecodeLeafData(data []byte) (*big.Int, []gameTypes.Leaf, error)
}

type LeafFetcher struct {
	log    log.Logger
	source L1Source
}

func (f *LeafFetcher) FetchLeaves(ctx context.Context, blockHash common.Hash, oracle Oracle, ident gameTypes.LargePreimageIdent) ([]gameTypes.Leaf, error) {
	blockNums, err := oracle.GetLeafBlocks(ctx, batching.BlockByHash(blockHash), ident)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve leaf block nums: %w", err)
	}
	chainID, err := f.source.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L1 chain ID: %w", err)
	}
	signer := types.LatestSignerForChainID(chainID)
	var leaves []gameTypes.Leaf
	for _, blockNum := range blockNums {
		foundRelevantTx := false
		block, err := f.source.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		if err != nil {
			return nil, fmt.Errorf("failed getting tx for block %v: %w", blockNum, err)
		}
		for _, tx := range block.Transactions() {
			txLeaves, err := f.extractRelevantLeavesFromTx(ctx, oracle, signer, tx, ident)
			if err != nil {
				return nil, err
			}
			foundRelevantTx = foundRelevantTx || len(txLeaves) > 0
			leaves = append(leaves, txLeaves...)
		}
		if !foundRelevantTx {
			// The contract said there was a relevant transaction in this block but we failed to find out
			// There was either a reorg or the extraction logic is broken.
			// Either way, abort this attempt to validate the preimage
			return nil, fmt.Errorf("%w %v", ErrNoLeavesFound, blockNum)
		}
	}
	return leaves, nil
}

func (f *LeafFetcher) extractRelevantLeavesFromTx(ctx context.Context, oracle Oracle, signer types.Signer, tx *types.Transaction, ident gameTypes.LargePreimageIdent) ([]gameTypes.Leaf, error) {
	if tx.To() == nil || *tx.To() != oracle.Addr() {
		f.log.Trace("Skip tx with incorrect to addr", "tx", tx.Hash(), "expected", oracle.Addr(), "actual", tx.To())
		return nil, nil
	}
	uuid, txLeaves, err := oracle.DecodeLeafData(tx.Data())
	if errors.Is(err, contracts.ErrInvalidAddLeavesCall) {
		f.log.Trace("Skip tx with invalid call data", "tx", tx.Hash(), "err", err)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if uuid.Cmp(ident.UUID) != 0 {
		f.log.Trace("Skip tx with incorrect UUID", "tx", tx.Hash(), "expected", ident.UUID, "actual", uuid)
		return nil, nil
	}
	sender, err := signer.Sender(tx)
	if err != nil {
		f.log.Trace("Skipping transaction with invalid sender", "tx", tx.Hash(), "err", err)
		return nil, nil
	}
	if sender != ident.Claimant {
		f.log.Trace("Skipping transaction with incorrect sender", "tx", tx.Hash(), "expected", ident.Claimant, "actual", sender)
		return nil, nil
	}
	rcpt, err := f.source.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve receipt for tx %v: %w", tx.Hash(), err)
	}
	if rcpt.Status != types.ReceiptStatusSuccessful {
		f.log.Trace("Skipping transaction with failed receipt status", "tx", tx.Hash(), "status", rcpt.Status)
		return nil, nil
	}
	return txLeaves, nil
}

func NewPreimageFetcher(logger log.Logger, source L1Source) *LeafFetcher {
	return &LeafFetcher{
		log:    logger,
		source: source,
	}
}
