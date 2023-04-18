package l2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	cll2 "github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func NewEngine(logger log.Logger, pre preimage.Oracle, hint preimage.Hinter, cfg *config.Config) (*cll2.OracleEngine, error) {
	oracle := cll2.NewCachingOracle(cll2.NewPreimageOracle(pre, hint))
	genesis, err := loadL2Genesis(cfg)
	if err != nil {
		return nil, err
	}
	engineBackend, err := cll2.NewOracleBackedL2Chain(logger, oracle, genesis, cfg.L2Head)
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

func loadL2Genesis(cfg *config.Config) (*params.ChainConfig, error) {
	data, err := os.ReadFile(cfg.L2GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("read l2 genesis file: %w", err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, fmt.Errorf("parse l2 genesis file: %w", err)
	}
	return genesis.Config, nil
}
