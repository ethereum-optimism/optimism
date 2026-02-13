package fetcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNoLeavesFound = errors.New("no leaves found in block")
)

type L1Source interface {
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)
	apis.ReceiptsFetcher
}

type Oracle interface {
	Addr() common.Address
	GetInputDataBlocks(ctx context.Context, block rpcblock.Block, ident keccakTypes.LargePreimageIdent) ([]uint64, error)
	DecodeInputData(data []byte) (*big.Int, keccakTypes.InputData, error)
}

type InputFetcher struct {
	log    log.Logger
	source L1Source
}

func (f *InputFetcher) FetchInputs(ctx context.Context, blockHash common.Hash, oracle Oracle, ident keccakTypes.LargePreimageIdent) ([]keccakTypes.InputData, error) {
	blockNums, err := oracle.GetInputDataBlocks(ctx, rpcblock.ByHash(blockHash), ident)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve leaf block nums: %w", err)
	}
	var inputs []keccakTypes.InputData
	for _, blockNum := range blockNums {
		foundRelevantTx := false
		blockRef, err := f.source.BlockRefByNumber(ctx, blockNum)
		if err != nil {
			return nil, fmt.Errorf("failed getting info for block %v: %w", blockNum, err)
		}
		_, receipts, err := f.source.FetchReceipts(ctx, blockRef.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve receipts for block %v: %w", blockNum, err)
		}
		for _, rcpt := range receipts {
			inputData, err := f.extractRelevantLeavesFromReceipt(rcpt, oracle, ident)
			if err != nil {
				return nil, err
			}
			if len(inputData) > 0 {
				foundRelevantTx = true
				inputs = append(inputs, inputData...)
			}
		}
		if !foundRelevantTx {
			// The contract said there was a relevant transaction in this block that we failed to find.
			// There was either a reorg or the extraction logic is broken.
			// Either way, abort this attempt to validate the preimage.
			return nil, fmt.Errorf("%w %v", ErrNoLeavesFound, blockNum)
		}
	}
	return inputs, nil
}

func (f *InputFetcher) extractRelevantLeavesFromReceipt(rcpt *types.Receipt, oracle Oracle, ident keccakTypes.LargePreimageIdent) ([]keccakTypes.InputData, error) {
	if rcpt.Status != types.ReceiptStatusSuccessful {
		f.log.Trace("Skipping transaction with failed receipt status", "tx", rcpt.TxHash, "status", rcpt.Status)
		return nil, nil
	}

	// Iterate over the logs from in this receipt, looking for relevant logs emitted from the oracle contract
	var inputs []keccakTypes.InputData
	for i, txLog := range rcpt.Logs {
		if txLog.Address != oracle.Addr() {
			f.log.Trace("Skip tx log not emitted by the oracle contract", "tx", rcpt.TxHash, "logIndex", i, "targetContract", oracle.Addr(), "actualContract", txLog.Address)
			continue
		}
		if len(txLog.Data) < 20 {
			f.log.Trace("Skip tx log with insufficient data (less than 20 bytes)", "tx", rcpt.TxHash, "logIndex", i, "dataLength", len(txLog.Data))
			continue
		}
		caller := common.Address(txLog.Data[0:20])
		callData := txLog.Data[20:]

		if caller != ident.Claimant {
			f.log.Trace("Skip tx log from irrelevant claimant", "tx", rcpt.TxHash, "logIndex", i, "targetClaimant", ident.Claimant, "actualClaimant", caller)
			continue
		}
		uuid, inputData, err := oracle.DecodeInputData(callData)
		if errors.Is(err, contracts.ErrInvalidAddLeavesCall) {
			f.log.Trace("Skip tx log with call data not targeting expected method", "tx", rcpt.TxHash, "logIndex", i, "err", err)
			continue
		} else if err != nil {
			return nil, err
		}
		if uuid.Cmp(ident.UUID) != 0 {
			f.log.Trace("Skip tx log with irrelevant UUID", "tx", rcpt.TxHash, "logIndex", i, "targetUUID", ident.UUID, "actualUUID", uuid)
			continue
		}
		inputs = append(inputs, inputData)
	}

	return inputs, nil
}

func NewPreimageFetcher(logger log.Logger, source L1Source) *InputFetcher {
	return &InputFetcher{
		log:    logger,
		source: source,
	}
}
