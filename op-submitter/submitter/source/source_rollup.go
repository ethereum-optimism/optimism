package source

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var supportedL2OutputVersion = eth.Bytes32{}

type RollupSubmissionSource struct {
	provider dial.RollupProvider
}

func NewRollupSubmissionSource(provider dial.RollupProvider) *RollupSubmissionSource {
	return &RollupSubmissionSource{
		provider: provider,
	}
}

func (r *RollupSubmissionSource) Close() {
	r.provider.Close()
}

func (r *RollupSubmissionSource) SyncStatus(ctx context.Context) (SyncStatus, error) {
	client, err := r.provider.RollupClient(ctx)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("failed to select active rollup client: %w", err)
	}
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return SyncStatus{}, err
	}
	return SyncStatus{
		CurrentL1:   status.CurrentL1,
		SafeL2:      status.SafeL2.Number,
		FinalizedL2: status.FinalizedL2.Number,
	}, nil
}

func (r *RollupSubmissionSource) SubmissionAtSequenceNum(ctx context.Context, blockNum uint64) (Submission, error) {
	client, err := r.provider.RollupClient(ctx)
	if err != nil {
		return Submission{}, fmt.Errorf("failed to select active rollup client: %w", err)
	}
	output, err := client.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return Submission{}, err
	}

	if output.Version != supportedL2OutputVersion {
		return Submission{}, fmt.Errorf("unsupported l2 output version: %v, supported: %v", output.Version, supportedL2OutputVersion)
	}
	return Submission{
		Root:        common.Hash(output.OutputRoot),
		SequenceNum: output.BlockRef.Number,
		CurrentL1:   output.Status.CurrentL1.ID(),
	}, nil
}
