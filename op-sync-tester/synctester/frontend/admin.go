package frontend

import (
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type AdminBackend interface {
	ClearSessions(id sttypes.SyncTesterID)
}

type AdminFrontend struct {
	b AdminBackend
}

func NewAdminFrontend(b AdminBackend) *AdminFrontend {
	return &AdminFrontend{b: b}
}

func (s *AdminFrontend) ClearSessions(id sttypes.SyncTesterID) {
	s.b.ClearSessions(id)
}
