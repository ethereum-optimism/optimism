package service

import (
	"context"
	"op-ufm/pkg/config"
	"op-ufm/pkg/provider"

	"github.com/ethereum/go-ethereum/log"
)

type Service struct {
	Config    *config.Config
	Healthz   *Healthz
	Providers map[string]*provider.Provider
}

func New(cfg *config.Config) *Service {
	s := &Service{
		Config:    cfg,
		Healthz:   &Healthz{},
		Providers: make(map[string]*provider.Provider, len(cfg.Providers)),
	}
	return s
}

func (s *Service) Start(ctx context.Context) {
	log.Info("service starting")
	if s.Config.Healthz.Enabled {
		s.Healthz.Start(ctx, s.Config.Healthz.Host, s.Config.Healthz.Port)
		log.Info("healthz started")
	}

	// map networks to its providers
	networks := make(map[string][]string)
	for name, providerConfig := range s.Config.Providers {
		if providerConfig.Disabled {
			continue
		}
		networks[providerConfig.Network] = append(networks[providerConfig.Network], name)
	}

	txpool := &provider.TransactionPool{}
	for name, providers := range networks {
		if len(providers) == 1 {
			log.Warn("can't measure first seen for network, please another provider", "network", name)
		}
		(*txpool)[name] = &provider.NetworkTransactionPool{}
		(*txpool)[name].Transactions = make(map[string]*provider.TransactionState)
		(*txpool)[name].Expected = len(providers)
	}

	for name, providerConfig := range s.Config.Providers {
		if providerConfig.Disabled {
			log.Info("provider is disabled", "provider", name)
			continue
		}
		s.Providers[name] = provider.New(name,
			providerConfig,
			&s.Config.Signer,
			s.Config.Wallets[providerConfig.Wallet],
			(*txpool)[providerConfig.Network])
		s.Providers[name].Start(ctx)
		log.Info("provider started", "provider", name)
	}
	log.Info("service started")
}

func (s *Service) Shutdown() {
	log.Info("service shutting down")
	if s.Config.Healthz.Enabled {
		s.Healthz.Shutdown()
		log.Info("healthz stopped")
	}
	for name, provider := range s.Providers {
		provider.Shutdown()
		log.Info("provider stopped", "provider", name)
	}
	log.Info("service stopped")
}
