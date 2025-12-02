package source

import (
	"context"
	"encoding/binary"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type Proposal struct {
	// Root is the proposal hash
	Root common.Hash

	// SequenceNum represents the L2 Block number or Super Root L2 timestamp
	SequenceNum uint64

	// Super is present if, and only if, this Proposal is a Super Root proposal
	Super eth.Super

	CurrentL1 eth.BlockID

	// Legacy provides data that is only available when retrieving data from a single rollup node.
	// It should only be used for L2OO proposals.
	Legacy LegacyProposalData
}

// IsSuperRootProposal returns true if the proposal is a Super Root proposal.
func (p *Proposal) IsSuperRootProposal() bool {
	return p.Super != nil
}

// ExtraData returns the Dispute Game extra data as appropriate for the proposal type.
func (p *Proposal) ExtraData() []byte {
	if p.Super != nil {
		return p.Super.Marshal()
	} else {
		var extraData [32]byte
		binary.BigEndian.PutUint64(extraData[24:], p.SequenceNum)
		return extraData[:]
	}
}

type LegacyProposalData struct {
	HeadL1      eth.L1BlockRef
	SafeL2      eth.L2BlockRef
	FinalizedL2 eth.L2BlockRef

	// Support legacy metrics when possible
	BlockRef eth.L2BlockRef
}

type ProposalSource interface {
	ProposalAtSequenceNum(ctx context.Context, seqNum uint64) (Proposal, error)
	SyncStatus(ctx context.Context) (SyncStatus, error)

	// Close closes the underlying client or clients
	Close()
}

type SyncStatus struct {
	CurrentL1   eth.L1BlockRef
	SafeL2      uint64
	FinalizedL2 uint64
}
