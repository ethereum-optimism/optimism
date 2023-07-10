package op_challenger

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
)

// Main is the programmatic entry-point for running op-challenger
func Main(logger log.Logger, cfg *config.Config, metricsFactory opmetrics.Factory) error {
	logger.Info("Fault game started")
	return nil
}
