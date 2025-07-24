package frontend

import (
	"context"
)

type SyncBackend interface {
	GetSession(ctx context.Context) error
	DeleteSession(ctx context.Context) error
	ListSessions(ctx context.Context) ([]string, error)
}

type SyncFrontend struct {
	b SyncBackend
}

func NewSyncFrontend(b SyncBackend) *SyncFrontend {
	return &SyncFrontend{b: b}
}

func (s *SyncFrontend) GetSession(ctx context.Context) error {
	return s.b.GetSession(ctx)
}

func (s *SyncFrontend) DeleteSession(ctx context.Context) error {
	return s.b.DeleteSession(ctx)
}

func (s *SyncFrontend) ListSessions(ctx context.Context) ([]string, error) {
	return s.b.ListSessions(ctx)
}
