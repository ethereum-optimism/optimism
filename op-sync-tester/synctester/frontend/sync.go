package frontend

import (
	"context"
)

type SyncBackend interface {
	Init(ctx context.Context) error
}

type SyncFrontend struct {
	b SyncBackend
}

func NewSyncFrontend(b SyncBackend) *SyncFrontend {
	return &SyncFrontend{b: b}
}

func (s *SyncFrontend) Init(ctx context.Context) error {
	return s.b.Init(ctx)
}
