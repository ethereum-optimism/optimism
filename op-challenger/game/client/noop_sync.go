package client

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type NoopSyncStatusValidator struct{}

func (n *NoopSyncStatusValidator) ValidateNodeSynced(_ context.Context, _ eth.BlockID) error {
	return nil
}

var _ types.SyncValidator = (*NoopSyncStatusValidator)(nil)
