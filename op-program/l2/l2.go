package l2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-program/config"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func NewFetchingEngine(ctx context.Context, logger log.Logger, cfg *config.Config) (derive.Engine, error) {
	genesis, err := loadL2Genesis(cfg)
	if err != nil {
		return nil, err
	}
	oracle, err := NewFetchingL2Oracle(ctx, logger, cfg.L2URL)
	if err != nil {
		return nil, fmt.Errorf("connect l2 oracle: %w", err)
	}

	engineBackend, err := NewOracleBackedL2Chain(logger, oracle, genesis, cfg.L2Head)
	if err != nil {
		return nil, fmt.Errorf("create l2 chain: %w", err)
	}
	return NewOracleEngine(cfg.Rollup, logger, engineBackend), nil
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
