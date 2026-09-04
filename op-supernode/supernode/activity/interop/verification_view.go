package interop

import (
	"fmt"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

type frontierQueryKey struct {
	blockNum  uint64
	timestamp uint64
	logIdx    uint32
	checksum  messages.MessageChecksum
}

type frontierBlockView struct {
	ref      eth.BlockRef
	execMsgs map[uint32]*messages.ExecutingMessage
	contains map[frontierQueryKey]messages.BlockSeal
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
		if cc.IngestionSourceOf(chain) == cc.IngestionProven {
			// A proof-carried chain has no receipts to build a frontier view from — and needs
			// none. The frontier view exists because a DRIVEN chain's frontier block is not in its
			// message database yet: it is sealed later, when the round advances. A proven chain's
			// is sealed the moment the proof lands, which is always at or before the frontier, so
			// leaving it out of the view routes every question about it to the message database
			// instead — and that path yields the block's REAL wire hash and containment answered
			// from its sealed logs.
			//
			// Its EXECUTING messages come from neither: not from receipts, and not from the message
			// database (which keys them by log position, which the wire does not carry). They come
			// from the chain container, in verifyInteropMessages — the G7 judge flip. Before G7 they
			// were empty here and P's dependencies were proof-trusted; that posture is retired.
			//
			// One consequence is worth stating because it is load-bearing rather than incidental: a
			// proven chain never appears in this view, so it never supplies the same-timestamp
			// `contains` short-circuit. Nothing needs it to — its logs are already sealed — and its
			// own imports are forbidden from being same-timestamp anyway (G7G D2).
			continue
		}
		blockInfo, receipts, err := chain.FetchReceipts(i.ctx, blockID)
		if err != nil {
			return nil, fmt.Errorf("chain %s: failed to fetch receipts for frontier block %s: %w", chainID, blockID, err)
		}
		view.blocks[chainID] = buildFrontierBlockView(chainID, blockInfo, receipts.Geth())
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
	execMsgs := make(map[uint32]*messages.ExecutingMessage)
	contains := make(map[frontierQueryKey]messages.BlockSeal)

	var logIdx uint32
	for _, receipt := range receipts {
		for _, entry := range receipt.Logs {
			logHash := messages.LogToLogHash(entry)
			query := messages.ChecksumArgs{
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
			}] = messages.BlockSeal{
				Hash:      ref.Hash,
				Number:    ref.Number,
				Timestamp: ref.Time,
			}

			if execMsg, err := messages.DecodeExecutingMessageLog(entry); err == nil && execMsg != nil {
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

func (v *frontierVerificationView) contains(chainID eth.ChainID, query messages.ContainsQuery) (messages.BlockSeal, bool) {
	block, ok := v.block(chainID)
	if !ok {
		return messages.BlockSeal{}, false
	}
	seal, ok := block.contains[frontierQueryKey{
		blockNum:  query.BlockNum,
		timestamp: query.Timestamp,
		logIdx:    query.LogIdx,
		checksum:  query.Checksum,
	}]
	return seal, ok
}
