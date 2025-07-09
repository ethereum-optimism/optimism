package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
)

type SyncTesterService struct {
	service *synctester.Service
}

func (s *SyncTesterService) hydrate(system stack.ExtensibleSystem) {
}
