package proofbatch

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ExportBlock converts the information the normal batcher already loads for one L2 block into the
// transaction-stripped proof-batch representation. Keeping this here gives the production batcher,
// sysgo controls, and fixtures one canonical answer.
func ExportBlock(payload *eth.ExecutionPayload, output *eth.OutputResponse, receipts optypes.Receipts) (BlockExport, error) {
	if payload == nil || output == nil {
		return BlockExport{}, fmt.Errorf("payload and output are required")
	}
	number := uint64(payload.BlockNumber)
	if output.BlockRef.Number != number || output.BlockRef.Hash != payload.BlockHash || output.BlockRef.Time != uint64(payload.Timestamp) {
		return BlockExport{}, fmt.Errorf("rollup output for block %d does not describe payload %s", number, payload.ID())
	}
	if output.StateRoot != common.Hash(payload.StateRoot) {
		return BlockExport{}, fmt.Errorf("block %d output state root %s differs from payload %s",
			number, output.StateRoot, payload.StateRoot)
	}
	if payload.WithdrawalsRoot == nil {
		return BlockExport{}, fmt.Errorf("block %d has no Isthmus message-passer root", number)
	}
	if output.WithdrawalStorageRoot != *payload.WithdrawalsRoot {
		return BlockExport{}, fmt.Errorf("block %d output message-passer root %s differs from payload %s",
			number, output.WithdrawalStorageRoot, *payload.WithdrawalsRoot)
	}

	export := BlockExport{
		Number:                   number,
		Timestamp:                uint64(payload.Timestamp),
		Hash:                     payload.BlockHash,
		StateRoot:                common.Hash(payload.StateRoot),
		MessagePasserStorageRoot: *payload.WithdrawalsRoot,
		L1Origin:                 output.BlockRef.L1Origin,
		SequenceNumber:           output.BlockRef.SequenceNumber,
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		if receipt == nil {
			return BlockExport{}, fmt.Errorf("block %d has a nil receipt", number)
		}
		for _, entry := range receipt.Logs {
			if entry == nil {
				return BlockExport{}, fmt.Errorf("block %d has a nil log", number)
			}
			logs = append(logs, entry)
			export.Logs = append(export.Logs, LogExport{
				Index: uint32(entry.Index),
				Hash:  messages.LogToLogHash(entry),
			})
		}
	}
	var err error
	export.ExecMsgs, err = ExecMsgsFromLogs(logs)
	if err != nil {
		return BlockExport{}, fmt.Errorf("extract executing-message imports from L2 block %d: %w", number, err)
	}
	if got := common.Hash(export.OutputRoot()); got != common.Hash(output.OutputRoot) {
		return BlockExport{}, fmt.Errorf("block %d proof export derives output root %s, rollup node reports %s",
			number, got, output.OutputRoot)
	}
	return export, nil
}
