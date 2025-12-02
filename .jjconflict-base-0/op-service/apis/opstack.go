package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

const (
	BuildErrCodeTemporary      = -40100
	BuildErrCodePrestate       = -40101
	BuildErrCodeInvalidInput   = -40110
	BuildErrCodeUnknownPayload = -40120
	BuildErrCodeOther          = -40199
)

type BuildAPI interface {
	// OpenBlock starts a block-building job with the given attributes.
	// The identifier of the job is returned, if successfully started.
	OpenBlock(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error)
	// CancelBlock cancels block-building.
	CancelBlock(ctx context.Context, id eth.PayloadInfo) error
	// SealBlock completes block-building. The block will not be canonical until committed to by CommitBlock
	SealBlock(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error)
}

type CommitAPI interface {
	// CommitBlock processes the block, and sets it as canonical block of the chain.
	CommitBlock(ctx context.Context, envelope *opsigner.SignedExecutionPayloadEnvelope) error
}

type PublishAPI interface {
	PublishBlock(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error
}

type OPStackAPI interface {
	BuildAPI
	CommitAPI
	PublishAPI
}
