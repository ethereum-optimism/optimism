package l2

import (
	"context"
	"fmt"

	cll2 "github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/log"
)

func NewEngine(logger log.Logger, pre preimage.Oracle, hint preimage.Hinter, cfg *config.Config) (*cll2.OracleEngine, error) {
	oracle := cll2.NewCachingOracle(cll2.NewPreimageOracle(pre, hint))
	engineBackend, err := cll2.NewOracleBackedL2Chain(logger, oracle, cfg.L2ChainConfig, cfg.L2Head)
	if err != nil {
		return nil, fmt.Errorf("create l2 chain: %w", err)
	}
	return cll2.NewOracleEngine(cfg.Rollup, logger, engineBackend), nil
}

func NewFetchingOracle(ctx context.Context, logger log.Logger, cfg *config.Config) (cll2.Oracle, error) {
	oracle, err := NewFetchingL2Oracle(ctx, logger, cfg.L2URL, cfg.L2Head)
	if err != nil {
		return nil, fmt.Errorf("connect l2 oracle: %w", err)
	}
	return oracle, nil
}
