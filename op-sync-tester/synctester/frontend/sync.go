package frontend

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncBackend interface {
	ChainID(ctx context.Context) (eth.ChainID, error)
	Init(ctx context.Context) (string, error)
}

type SyncFrontend struct {
	b SyncBackend
}

var _ apis.SyncTester = (*SyncFrontend)(nil)

func NewSyncFrontend(b SyncBackend) *SyncFrontend {
	return &SyncFrontend{b: b}
}

func (s *SyncFrontend) ChainID(ctx context.Context) (eth.ChainID, error) {
	return s.b.ChainID(ctx)
}

func (s *SyncFrontend) Init(ctx context.Context) (string, error) {
	return s.b.Init(ctx)
}
