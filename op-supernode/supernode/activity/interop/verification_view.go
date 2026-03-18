package interop

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

type frontierQueryKey struct {
	blockNum  uint64
	timestamp uint64
	logIdx    uint32
	checksum  suptypes.MessageChecksum
}

type frontierBlockView struct {
	ref      eth.BlockRef
	execMsgs map[uint32]*suptypes.ExecutingMessage
	contains map[frontierQueryKey]suptypes.BlockSeal
}

type frontierVerificationView struct {
	blocks map[eth.ChainID]frontierBlockView
}

func (i *Interop) resolveFrontierVerificationView(blocksAtTS map[eth.ChainID]eth.BlockID) (*frontierVerificationView, error) {
	view := &frontierVerificationView{
		blocks: make(map[eth.ChainID]frontierBlockView, len(blocksAtTS)),
	}
	for chainID, blockID := range blocksAtTS {
		chain, ok := i.chains[chainID]
		if !ok {
			continue
		}
		blockInfo, receipts, err := chain.FetchReceipts(i.ctx, blockID)
		if err != nil {
			return nil, fmt.Errorf("chain %s: failed to fetch receipts for frontier block %s: %w", chainID, blockID, err)
		}
		view.blocks[chainID] = buildFrontierBlockView(chainID, blockInfo, receipts)
	}
	return view, nil
}

func buildFrontierBlockView(chainID eth.ChainID, blockInfo eth.BlockInfo, receipts gethTypes.Receipts) frontierBlockView {
	ref := eth.BlockRef{
		Hash:       blockInfo.Hash(),
		Number:     blockInfo.NumberU64(),
		ParentHash: blockInfo.ParentHash(),
		Time:       blockInfo.Time(),
	}
	execMsgs := make(map[uint32]*suptypes.ExecutingMessage)
	contains := make(map[frontierQueryKey]suptypes.BlockSeal)

	var logIdx uint32
	for _, receipt := range receipts {
		for _, entry := range receipt.Logs {
			logHash := processors.LogToLogHash(entry)
			query := suptypes.ChecksumArgs{
				BlockNumber: ref.Number,
				LogIndex:    logIdx,
				Timestamp:   ref.Time,
				ChainID:     chainID,
				LogHash:     logHash,
			}.Query()
			contains[frontierQueryKey{
				blockNum:  query.BlockNum,
				timestamp: query.Timestamp,
				logIdx:    query.LogIdx,
				checksum:  query.Checksum,
			}] = suptypes.BlockSeal{
				Hash:      ref.Hash,
				Number:    ref.Number,
				Timestamp: ref.Time,
			}

			if execMsg, err := processors.DecodeExecutingMessageLog(entry); err == nil && execMsg != nil {
				execMsgs[logIdx] = execMsg
			}
			logIdx++
		}
	}

	return frontierBlockView{
		ref:      ref,
		execMsgs: execMsgs,
		contains: contains,
	}
}

func (v *frontierVerificationView) block(chainID eth.ChainID) (frontierBlockView, bool) {
	if v == nil {
		return frontierBlockView{}, false
	}
	block, ok := v.blocks[chainID]
	return block, ok
}

func (v *frontierVerificationView) contains(chainID eth.ChainID, query suptypes.ContainsQuery) (suptypes.BlockSeal, bool) {
	block, ok := v.block(chainID)
	if !ok {
		return suptypes.BlockSeal{}, false
	}
	seal, ok := block.contains[frontierQueryKey{
		blockNum:  query.BlockNum,
		timestamp: query.Timestamp,
		logIdx:    query.LogIdx,
		checksum:  query.Checksum,
	}]
	return seal, ok
}
