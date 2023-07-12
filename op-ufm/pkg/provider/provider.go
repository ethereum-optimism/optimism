package provider

import (
	"context"
	"net/http"
	"op-ufm/pkg/config"
	"time"
)

type Provider struct {
	name         string
	config       *config.ProviderConfig
	signerConfig *config.SignerServiceConfig
	walletConfig *config.WalletConfig
	txPool       *NetworkTransactionPool
	cancelFunc   context.CancelFunc

	client *http.Client
}

func New(name string, cfg *config.ProviderConfig,
	signerConfig *config.SignerServiceConfig,
	walletConfig *config.WalletConfig,
	txPool *NetworkTransactionPool) *Provider {
	p := &Provider{
		name:         name,
		config:       cfg,
		signerConfig: signerConfig,
		walletConfig: walletConfig,
		txPool:       txPool,

		client: http.DefaultClient,
	}
	return p
}

func (p *Provider) Start(ctx context.Context) {
	providerCtx, cancelFunc := context.WithCancel(ctx)
	p.cancelFunc = cancelFunc
	schedule(providerCtx, time.Duration(p.config.ReadInterval), p.Heartbeat)
	if !p.config.ReadOnly {
		schedule(providerCtx, time.Duration(p.config.SendInterval), p.Roundtrip)
	}
}

func (p *Provider) Shutdown() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
}

func schedule(ctx context.Context, interval time.Duration, handler func(ctx context.Context)) {
	go func() {
		for {
			timer := time.NewTimer(interval)
			handler(ctx)

			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}
