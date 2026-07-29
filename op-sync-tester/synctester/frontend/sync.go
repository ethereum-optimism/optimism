package frontend

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncBackend interface {
	apis.SyncAPI
}
type SyncFrontend struct {
	b SyncBackend
}

func NewSyncFrontend(b SyncBackend) *SyncFrontend {
	return &SyncFrontend{b: b}
}

func (s *SyncFrontend) GetSession(ctx context.Context) (*eth.SyncTesterSession, error) {
	return s.b.GetSession(ctx)
}

func (s *SyncFrontend) DeleteSession(ctx context.Context) error {
	return s.b.DeleteSession(ctx)
}

func (s *SyncFrontend) ListSessions(ctx context.Context) ([]string, error) {
	return s.b.ListSessions(ctx)
}

func (s *SyncFrontend) ResetSession(ctx context.Context) error {
	return s.b.ResetSession(ctx)
}
