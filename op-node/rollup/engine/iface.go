package engine

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Engine interface {
	ExecEngine
	derive.L2Source
}

// CLSyncEngine provides the core engine state interface plus unsafe payload insertion
type CLSyncEngine interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	// L2ChainState returns all three L2 heads in a single call for efficiency
	L2ChainState() eth.L2ChainState
	InsertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error
}

type LocalEngineState interface {
	CLSyncEngine

	PendingSafeL2Head() eth.L2BlockRef
	BackupUnsafeL2Head() eth.L2BlockRef
}

type LocalEngineControl interface {
	LocalEngineState
	ResetEngineControl
}

var _ LocalEngineControl = (*EngineController)(nil)
var _ CLSyncEngine = (*EngineController)(nil)
