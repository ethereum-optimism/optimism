package supernode

import (
	"context"
	"sync"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supernode/config"
)

type Supernode struct {
	log         gethlog.Logger
	version     string
	requestStop context.CancelCauseFunc
	stopped     bool
	cfg         *config.CLIConfig
	chains      map[types.ChainID]cc.ChainContainer
	wg          sync.WaitGroup
}

func New(log gethlog.Logger, version string, requestStop context.CancelCauseFunc, cfg *config.CLIConfig, vnCfgs map[types.ChainID]*opnodecfg.Config) *Supernode {
	s := &Supernode{log: log, version: version, requestStop: requestStop, cfg: cfg, chains: make(map[types.ChainID]cc.ChainContainer)}
	// Initialize chain containers for each configured chain ID
	for _, id := range cfg.Chains {
		if vnCfgs[types.ChainID(id)] == nil {
			log.Error("missing virtual node config for chain", "chain", id)
			continue
		}
		chainID := types.ChainID(id)
		s.chains[chainID] = cc.NewChainContainer(chainID, vnCfgs[chainID], log, *cfg)
	}
	return s
}

func (s *Supernode) Start(ctx context.Context) error {
	s.log.Info("supernode starting", "version", s.version, "sample", s.cfg.Sample)
	// start all chain containers in their own routines
	for _, chain := range s.chains {
		s.wg.Add(1)
		go func(chain cc.ChainContainer) {
			defer s.wg.Done()
			if err := chain.Start(ctx); err != nil {
				s.log.Error("error starting chain", "chain", chain, "error", err)
			}
		}(chain)
	}
	// Block until context is cancelled (e.g., by interrupt signal)
	<-ctx.Done()
	s.log.Info("supernode received stop signal")
	return ctx.Err()
}

func (s *Supernode) Stop(ctx context.Context) error {
	s.log.Info("supernode exiting")
	s.stopped = true
	// stop all chain containers
	for _, chain := range s.chains {
		chain.Stop(ctx)
	}
	return nil
}

func (s *Supernode) Stopped() bool { return s.stopped }
