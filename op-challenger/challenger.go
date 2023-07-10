package op_challenger

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum/go-ethereum/log"
)

// Main is the programmatic entry-point for running op-challenger
func Main(logger log.Logger, cfg *config.Config) error {
	logger.Info("Fault game started")
	return nil
}
