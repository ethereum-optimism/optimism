package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type Submission struct {
	// Root is the proposal hash
	Root common.Hash
	// SequenceNum identifies the position in the overall state transition.
	// For output roots this is the L2 block number.
	// For super roots this is the timestamp.
	SequenceNum uint64
	CurrentL1   eth.BlockID
}

type SubmissionSource interface {
	ProposalAtSequenceNum(ctx context.Context, seqNum uint64) (Submission, error)
	SyncStatus(ctx context.Context) (SyncStatus, error)

	// Close closes the underlying client or clients
	Close()
}

type SyncStatus struct {
	CurrentL1   eth.L1BlockRef
	SafeL2      uint64
	FinalizedL2 uint64
}
