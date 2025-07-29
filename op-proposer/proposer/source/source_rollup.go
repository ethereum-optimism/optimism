package source

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var supportedL2OutputVersion = eth.Bytes32{}

// RollupProposalSource encapsulates a rollup provider for L2OO & pre-interop DGF.
type RollupProposalSource struct {
	provider dial.RollupProvider
}

// NewRollupProposalSource constructs a new `RollupProposalSource`.
func NewRollupProposalSource(provider dial.RollupProvider) *RollupProposalSource {
	return &RollupProposalSource{
		provider: provider,
	}
}

// Close closes the underlying rollup provider.
func (r *RollupProposalSource) Close() {
	r.provider.Close()
}

// SyncStatus returns the current L1 block, safe L2 block number, and finalized L2 block number.
//
// Errors if the provider fails to select an active rollup client or if the rollup client fails to return a sync status.
func (r *RollupProposalSource) SyncStatus(ctx context.Context) (SyncStatus, error) {
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

// ProposalAtSequenceNum returns the proposal data for the given sequence number.
//
// Errors if:
//
//   - The provider fails to select an active rollup client.
//   - The rollup client fails to return an output.
//   - The output is for an unsupported version.
func (r *RollupProposalSource) ProposalAtSequenceNum(ctx context.Context, blockNum uint64) (Proposal, error) {
	client, err := r.provider.RollupClient(ctx)
	if err != nil {
		return Proposal{}, fmt.Errorf("failed to select active rollup client: %w", err)
	}
	output, err := client.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return Proposal{}, err
	}

	if output.Version != supportedL2OutputVersion {
		return Proposal{}, fmt.Errorf("unsupported l2 output version: %v, supported: %v", output.Version, supportedL2OutputVersion)
	}
	return Proposal{
		Root:        common.Hash(output.OutputRoot),
		SequenceNum: output.BlockRef.Number,
		CurrentL1:   output.Status.CurrentL1.ID(),
		Legacy: LegacyProposalData{
			HeadL1:      output.Status.HeadL1,
			SafeL2:      output.Status.SafeL2,
			FinalizedL2: output.Status.FinalizedL2,
			BlockRef:    output.BlockRef,
		},
	}, nil
}
