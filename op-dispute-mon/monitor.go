package monitor

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/config"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

func Main(ctx context.Context, logger log.Logger, cfg *config.Config, options ...mon.ServiceOption) (*mon.Service, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return mon.NewService(ctx, logger, cfg, options...)
}
