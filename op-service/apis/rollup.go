package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type RollupConfig interface {
	RollupConfig(ctx context.Context) (*rollup.Config, error)
}

type DependencySetConfig interface {
	DependencySet(ctx context.Context) (depset.DependencySet, error)
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

type SdmStatus struct {
	PostExecOptIn  bool    `json:"postExecOptIn"`
	ProtocolActive bool    `json:"protocolActive"`
	Effective      bool    `json:"effective"`
	ActivationTime *uint64 `json:"activationTime,omitempty"`
}

type SequencerActivity interface {
	StartSequencer(ctx context.Context, unsafeHead common.Hash) error
	StopSequencer(ctx context.Context) (common.Hash, error)
	SequencerActive(ctx context.Context) (bool, error)
}

type SequencerSdmControl interface {
	SetSdmPostExecOptIn(ctx context.Context, enabled bool) error
	SdmStatus(ctx context.Context) (SdmStatus, error)
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
	SequencerSdmControl
	UnsignedPayloadPoster
	RollupConductor
	RecoverMode
}

type RollupAdminServer interface {
	CommonAdminServer
	SequencerActivity
	SequencerSdmControl
	UnsignedPayloadPoster
	RollupConductor
	RecoverMode
}

type RollupNodeClient interface {
	Version
	RollupConfig
	DependencySetConfig
	RollupSyncStatus
	RollupOutputClient
	RollupSafeAtClient
}

type RollupNodeServer interface {
	Version
	RollupConfig
	DependencySetConfig
	RollupSyncStatus
	RollupOutputServer
	RollupSafeAtServer
}

type RollupClient interface {
	RollupAdminClient
	RollupNodeClient
}
