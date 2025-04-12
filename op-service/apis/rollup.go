package apis

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type RollupConfig interface {
	RollupConfig(ctx context.Context) (*rollup.Config, error)
}

type RollupSyncStatus interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

type RollupOutputClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type RollupOutputServer interface {
	OutputAtBlock(ctx context.Context, blockNum hexutil.Uint64) (*eth.OutputResponse, error)
}

type RollupSafeAtClient interface {
	SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error)
}

type RollupSafeAtServer interface {
	SafeHeadAtL1Block(ctx context.Context, blockNum hexutil.Uint64) (*eth.SafeHeadResponse, error)
}

type SequencerActivity interface {
	StartSequencer(ctx context.Context, unsafeHead common.Hash) error
	StopSequencer(ctx context.Context) (common.Hash, error)
	SequencerActive(ctx context.Context) (bool, error)
}

type UnsignedPayloadPoster interface {
	PostUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
}

type RollupConductor interface {
	OverrideLeader(ctx context.Context) error
	ConductorEnabled(ctx context.Context) (bool, error)
}

type RecoverMode interface {
	SetRecoverMode(ctx context.Context, mode bool) error
}

type RollupAdminClient interface {
	CommonAdminClient
	SequencerActivity
	UnsignedPayloadPoster
	RollupConductor
	RecoverMode
}

type RollupAdminServer interface {
	CommonAdminServer
	SequencerActivity
	UnsignedPayloadPoster
	RollupConductor
	RecoverMode
}

type RollupNodeClient interface {
	Version
	RollupConfig
	RollupSyncStatus
	RollupOutputClient
	RollupSafeAtClient
}

type RollupNodeServer interface {
	Version
	RollupConfig
	RollupSyncStatus
	RollupOutputServer
	RollupSafeAtServer
}

type RollupClient interface {
	RollupAdminClient
	RollupNodeClient
}
