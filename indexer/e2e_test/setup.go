package e2e_tests

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processor"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

type E2ETestSuite struct {
	t *testing.T

	// Indexer
	DB      *database.DB
	Indexer *indexer.Indexer

	// Rollup
	OpCfg *op_e2e.SystemConfig
	OpSys *op_e2e.System
}

func createE2ETestSuite(t *testing.T) E2ETestSuite {
	dbUser := os.Getenv("DB_USER")
	dbName := setupTestDatabase(t)

	// Replace the handler of the global logger with the testlog
	logger := testlog.Logger(t, log.LvlInfo)
	log.Root().SetHandler(logger.GetHandler())

	// Rollup System Configuration and Start
	opCfg := op_e2e.DefaultSystemConfig(t)
	opSys, err := opCfg.Start()
	require.NoError(t, err)

	l1Contracts := processor.L1Contracts{
		OptimismPortal:         opCfg.L1Deployments.OptimismPortalProxy,
		L2OutputOracle:         opCfg.L1Deployments.L2OutputOracleProxy,
		L1CrossDomainMessenger: opCfg.L1Deployments.L1CrossDomainMessengerProxy,
		L1StandardBridge:       opCfg.L1Deployments.L1StandardBridgeProxy,
		L1ERC721Bridge:         opCfg.L1Deployments.L1ERC721BridgeProxy,
	}

	// Indexer Configuration and Start
	indexerCfg := config.Config{
		DB: config.DBConfig{
			Host: "127.0.0.1",
			Port: 5432,
			Name: dbName,
			User: dbUser,
		},
		RPCs: config.RPCsConfig{
			L1RPC: opSys.Nodes["l1"].HTTPEndpoint(),
			L2RPC: opSys.Nodes["sequencer"].HTTPEndpoint(),
		},
		Logger: logger,
		Chain: config.ChainConfig{
			L1Contracts: l1Contracts,
		},
	}

	db, err := database.NewDB(fmt.Sprintf("postgres://%s@localhost:5432/%s?sslmode=disable", dbUser, dbName))
	require.NoError(t, err)
	indexer, err := indexer.NewIndexer(indexerCfg)
	require.NoError(t, err)

	indexerCtx, indexerStop := context.WithCancel(context.Background())
	go func() {
		err := indexer.Run(indexerCtx)
		require.NoError(t, err)

		indexer.Cleanup()
	}()

	t.Cleanup(func() {
		indexerStop()

		// wait a second for the stop signal to be received
		time.Sleep(1 * time.Second)
		indexer.Cleanup()

		db.Close()
		opSys.Close()
	})

	return E2ETestSuite{
		t:       t,
		DB:      db,
		Indexer: indexer,
		OpCfg:   &opCfg,
		OpSys:   opSys,
	}
}

func setupTestDatabase(t *testing.T) string {
	user := os.Getenv("DB_USER")
	pg, err := sql.Open("pgx", fmt.Sprintf("postgres://%s@localhost:5432?sslmode=disable", user))
	require.NoError(t, err)
	require.NoError(t, pg.Ping())

	// create database
	dbName := fmt.Sprintf("indexer_test_%d", time.Now().UnixNano())
	_, err = pg.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := pg.Exec("DROP DATABASE " + dbName)
		require.NoError(t, err)
		pg.Close()
	})

	// setup schema, migration files ware walked in lexical order
	t.Logf("created database %s", dbName)
	db, err := sql.Open("pgx", fmt.Sprintf("postgres://%s@localhost:5432/%s?sslmode=disable", user, dbName))
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	defer db.Close()

	t.Logf("running schema migrations...")
	require.NoError(t, filepath.Walk("../migrations", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() {
			return nil
		}

		t.Logf("running schema migration: %s", path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = db.Exec(string(data))
		return err
	}))

	t.Logf("schema loaded")
	return dbName
}
