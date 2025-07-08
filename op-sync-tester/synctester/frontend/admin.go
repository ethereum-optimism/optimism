package frontend

import "context"

type AdminBackend interface {
	ClearSessions(context.Context)
}

type AdminFrontend struct {
	b AdminBackend
}

func NewAdminFrontend(b AdminBackend) *AdminFrontend {
	return &AdminFrontend{b: b}
}

func (s *AdminFrontend) ClearSessions(ctx context.Context) {
	s.b.ClearSessions(ctx)
}
