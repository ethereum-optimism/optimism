package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
)

// Caveat: id is binded by a single EL(chainID), but service can support multiple ELs
type SyncTesterService struct {
	service *synctester.Service
}

func (n *SyncTesterService) DefaultEndpoint(chainID eth.ChainID) (string, string, bool) {
	if n == nil || n.service == nil {
		return "", "", false
	}
	for syncTesterID, mappedChainID := range n.service.SyncTesters() {
		if mappedChainID != chainID {
			continue
		}
		return syncTesterID.String(), n.service.SyncTesterRPC(chainID, false), true
	}
	return "", "", false
}

func (n *SyncTesterService) RPC() string {
	if n == nil || n.service == nil {
		return ""
	}
	return n.service.RPC()
}

func (n *SyncTesterService) SyncTesterRPCPath(chainID eth.ChainID, withSessionID bool) string {
	if n == nil || n.service == nil {
		return ""
	}
	return n.service.SyncTesterRPCPath(chainID, withSessionID)
}
