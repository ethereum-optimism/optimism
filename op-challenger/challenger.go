package op_challenger

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault"
	"github.com/ethereum/go-ethereum/log"
)

// Main is the programmatic entry-point for running op-challenger
func Main(ctx context.Context, logger log.Logger, cfg *config.Config) error {
	service, err := fault.NewService(ctx, logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create the fault service: %w", err)
	}

	return service.MonitorGame(ctx)
}
