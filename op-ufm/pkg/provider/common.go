package provider

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
)

func (p *Provider) dial(ctx context.Context) (*ethclient.Client, error) {
	return ethclient.Dial(p.config.URL)
}
