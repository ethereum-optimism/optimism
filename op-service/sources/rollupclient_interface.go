package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type RollupClientInterface interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	RollupConfig(ctx context.Context) (*rollup.Config, error)
	Version(ctx context.Context) (string, error)
	StartSequencer(ctx context.Context, unsafeHead common.Hash) error
	StopSequencer(ctx context.Context) (common.Hash, error)
	SequencerActive(ctx context.Context) (bool, error)
	Close()
}
