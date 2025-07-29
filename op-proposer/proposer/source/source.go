package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// Proposal is the proposal data for L2OO & DGF.
type Proposal struct {
	// Root is the proposal hash
	Root common.Hash
	// SequenceNum identifies the position in the overall state transition.
	// For output roots this is the L2 block number.
	// For super roots this is the timestamp.
	SequenceNum uint64
	CurrentL1   eth.BlockID

	// Legacy provides data that is only available when retrieving data from a single rollup node.
	// It should only be used for L2OO proposals.
	Legacy LegacyProposalData
}

// LegacyProposalData is the legacy proposal data for L2OO and pre-interop DGF.
type LegacyProposalData struct {
	HeadL1      eth.L1BlockRef
	SafeL2      eth.L2BlockRef
	FinalizedL2 eth.L2BlockRef

	// Support legacy metrics when possible
	BlockRef eth.L2BlockRef
}

// ProposalSource is the common interface for all proposal sources.
type ProposalSource interface {
	ProposalAtSequenceNum(ctx context.Context, seqNum uint64) (Proposal, error)
	SyncStatus(ctx context.Context) (SyncStatus, error)

	// Close closes the underlying client or clients
	Close()
}

// SyncStatus is the sync status for all proposal sources.
type SyncStatus struct {
	CurrentL1   eth.L1BlockRef
	SafeL2      uint64
	FinalizedL2 uint64
}
