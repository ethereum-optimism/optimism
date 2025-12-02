package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/frontend"

	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type APIRouter interface {
	AddRPC(route string) error
	AddAPIToRPC(route string, api rpc.API) error
}

type Backend struct {
	log log.Logger
	m   metrics.Metricer

	syncTesters locks.RWMap[sttypes.SyncTesterID, *SyncTester]
}

func (b *Backend) Stop(ctx context.Context) error {
	// We have support for ctx/error here,
	// for future improvements like awaiting txs to complete and/or storing rate-limit data to disk.
	return nil
}

func FromConfig(log log.Logger, m metrics.Metricer, cfg *config.Config, router APIRouter) (*Backend, error) {
	b := &Backend{
		log: log,
		m:   m,
	}

	for stID, stCfg := range cfg.SyncTesters {
		st, err := SyncTesterFromConfig(log, m, stID, stCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to setup sync tester %q: %w", stID, err)
		}
		b.syncTesters.Set(stID, st)
	}
	// Set up the sync tester routes
	var syncTesterErr error
	b.syncTesters.Range(func(id sttypes.SyncTesterID, st *SyncTester) bool {
		path := "/chain/" + st.chainID.String() + "/synctest"
		if err := router.AddRPC(path); err != nil {
			syncTesterErr = errors.Join(fmt.Errorf("failed to set up synctest route: %w", err))
			return true
		}
		if err := router.AddAPIToRPC(path, rpc.API{
			Namespace: "sync",
			Service:   frontend.NewSyncFrontend(st),
		}); err != nil {
			syncTesterErr = errors.Join(syncTesterErr, fmt.Errorf("failed to add sync API: %w", err))
		}
		if err := router.AddAPIToRPC(path, rpc.API{
			Namespace: "eth",
			Service:   frontend.NewEthFrontend(st),
		}); err != nil {
			syncTesterErr = errors.Join(syncTesterErr, fmt.Errorf("failed to add eth API: %w", err))
		}
		if err := router.AddAPIToRPC(path, rpc.API{
			Namespace: "engine",
			Service:   frontend.NewEngineFrontend(st),
		}); err != nil {
			syncTesterErr = errors.Join(syncTesterErr, fmt.Errorf("failed to add engine API: %w", err))
		}
		return true
	})
	if syncTesterErr != nil {
		return nil, fmt.Errorf("failed to set up sync tester route(s): %w", syncTesterErr)
	}
	return b, nil
}

func (b *Backend) SyncTesters() (out map[sttypes.SyncTesterID]eth.ChainID) {
	out = make(map[sttypes.SyncTesterID]eth.ChainID)
	b.syncTesters.Range(func(key sttypes.SyncTesterID, value *SyncTester) bool {
		out[key] = value.chainID
		return true
	})
	return out
}
