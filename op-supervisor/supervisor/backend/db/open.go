package db

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"

	corelogs "github.com/ethereum-optimism/optimism/op-core/persistence/logsdb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/fromda"
)

func OpenLogDB(logger log.Logger, chainID eth.ChainID, dataDir string, m corelogs.Metrics) (LogStorage, error) {
	path, err := prepLogDBPath(chainID, dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create datadir for chain %s: %w", chainID, err)
	}
	logDB, err := corelogs.NewFromFile(logger, m, chainID, path, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create logdb for chain %s at %v: %w", chainID, path, err)
	}
	return NewCoreLogDBAdapter(logDB), nil
}

func OpenLocalDerivationDB(logger log.Logger, chainID eth.ChainID, dataDir string, m fromda.ChainMetrics) (*fromda.DB, error) {
	path, err := prepLocalDerivationDBPath(chainID, dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare datadir for chain %s: %w", chainID, err)
	}
	db, err := fromda.NewFromFile(logger, fromda.AdaptMetrics(m, "local_derived"), path)
	if err != nil {
		return nil, fmt.Errorf("failed to create local-derived for chain %s at %q: %w", chainID, path, err)
	}
	return db, nil
}

func OpenCrossDerivationDB(logger log.Logger, chainID eth.ChainID, dataDir string, m fromda.ChainMetrics) (*fromda.DB, error) {
	path, err := prepCrossDerivationDBPath(chainID, dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare datadir for chain %s: %w", chainID, err)
	}
	db, err := fromda.NewFromFile(logger, fromda.AdaptMetrics(m, "cross_derived"), path)
	if err != nil {
		return nil, fmt.Errorf("failed to create cross-derived for chain %s at %q: %w", chainID, path, err)
	}
	return db, nil
}
