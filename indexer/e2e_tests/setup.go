package e2e_tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

/*
	NOTE - Most of the current bridge tests fetch chain data via direct database queries. These could all
	be transitioned to use the API client instead to better simulate/validate real-world usage.
	Supporting this would potentially require adding new API endpoints for the specific query lookup types.
*/

type E2ETestSuite struct {
	t               *testing.T
	MetricsRegistry *prometheus.Registry

	// API
	API       *api.APIService
	ApiClient *api.Client

	// Indexer
	DB      *database.DB
	Indexer *indexer.Indexer

	// Rollup
	OpCfg *op_e2e.SystemConfig
	OpSys *op_e2e.System

	// Clients
	L1Client *ethclient.Client
	L2Client *ethclient.Client
}

type ConfigOpts func(*config.Config) *config.Config

func init() {
	// Disable the global logger. Ideally we'd like to dump geth
	// logs per-test but that's possible when running tests in
	// parallel as the root logger is shared.
	oplog.SetGlobalLogHandler(log.DiscardHandler())
}

// createE2ETestSuite ... Create a new E2E test suite
func createE2ETestSuite(t *testing.T, cfgOpt ...ConfigOpts) E2ETestSuite {
	dbUser := os.Getenv("DB_USER")
	dbName := setupTestDatabase(t)

	require.LessOrEqual(t, len(cfgOpt), 1)

	// E2E tests can run on the order of magnitude of minutes.
	// We mark the test as parallel before starting the devnet
	// to reduce that number of idle routines when paused.
	t.Parallel()

	opCfg := op_e2e.DefaultSystemConfig(t)

	// Unless specified, omit logs emitted by the various components
	if len(os.Getenv("ENABLE_ROLLUP_LOGS")) == 0 {
		t.Log("set env 'ENABLE_ROLLUP_LOGS' to show rollup logs")
		for name := range opCfg.Loggers {
			t.Logf("discarding logs for %s", name)
			noopLog := log.NewLogger(log.DiscardHandler())
			opCfg.Loggers[name] = noopLog
		}
	}

	// Rollup Start
	opSys, err := opCfg.Start(t)
	require.NoError(t, err)
	t.Cleanup(func() { opSys.Close() })

	// Indexer Configuration and Start
	indexerCfg := &config.Config{
		DB: config.DBConfig{Host: "127.0.0.1", Port: 5432, Name: dbName, User: dbUser},
		RPCs: config.RPCsConfig{
			L1RPC: opSys.EthInstances["l1"].HTTPEndpoint(),
			L2RPC: opSys.EthInstances["sequencer"].HTTPEndpoint(),
		},
		Chain: config.ChainConfig{
			L1PollingInterval: uint(opCfg.DeployConfig.L1BlockTime) * 1000,
			L2PollingInterval: uint(opCfg.DeployConfig.L2BlockTime) * 1000,
			L2Contracts:       config.L2ContractsFromPredeploys(),
			L1Contracts: config.L1Contracts{
				AddressManager:              opCfg.L1Deployments.AddressManager,
				SystemConfigProxy:           opCfg.L1Deployments.SystemConfigProxy,
				OptimismPortalProxy:         opCfg.L1Deployments.OptimismPortalProxy,
				L2OutputOracleProxy:         opCfg.L1Deployments.L2OutputOracleProxy,
				L1CrossDomainMessengerProxy: opCfg.L1Deployments.L1CrossDomainMessengerProxy,
				L1StandardBridgeProxy:       opCfg.L1Deployments.L1StandardBridgeProxy,
				L1ERC721BridgeProxy:         opCfg.L1Deployments.L1ERC721BridgeProxy,
				DisputeGameFactoryProxy:     opCfg.L1Deployments.DisputeGameFactoryProxy,
			},
		},
		HTTPServer:    config.ServerConfig{Host: "127.0.0.1", Port: 0},
		MetricsServer: config.ServerConfig{Host: "127.0.0.1", Port: 0},
	}

	// apply any settings
	for _, opt := range cfgOpt {
		indexerCfg = opt(indexerCfg)
	}

	indexerLog := testlog.Logger(t, log.LevelInfo).New("role", "indexer")
	ix, err := indexer.NewIndexer(context.Background(), indexerLog, indexerCfg, func(cause error) {
		if cause != nil {
			t.Fatalf("indexer shut down with critical error: %v", cause)
		}
	})
	require.NoError(t, err)
	require.NoError(t, ix.Start(context.Background()), "cleanly start indexer")
	t.Cleanup(func() { require.NoError(t, ix.Stop(context.Background())) })

	dbLog := testlog.Logger(t, log.LvlInfo).New("role", "db")
	db, err := database.NewDB(context.Background(), dbLog, indexerCfg.DB)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// API Configuration and Start
	apiLog := testlog.Logger(t, log.LevelInfo).New("role", "indexer_api")
	apiCfg := &api.Config{
		DB:            &api.TestDBConnector{BridgeTransfers: db.BridgeTransfers}, // reuse the same DB
		HTTPServer:    config.ServerConfig{Host: "127.0.0.1", Port: 0},
		MetricsServer: config.ServerConfig{Host: "127.0.0.1", Port: 0},
	}

	apiService, err := api.NewApi(context.Background(), apiLog, apiCfg)
	require.NoError(t, err, "create indexer API service")
	require.NoError(t, apiService.Start(context.Background()), "start indexer API service")
	t.Cleanup(func() {
		require.NoError(t, apiService.Stop(context.Background()), "cleanly shut down indexer")
	})

	// Wait for the API to start listening
	time.Sleep(1 * time.Second)

	apiClient, err := api.NewClient(&api.ClientConfig{PaginationLimit: 100, BaseURL: "http://" + apiService.Addr()})
	require.NoError(t, err, "must open indexer API client")

	return E2ETestSuite{
		t:               t,
		MetricsRegistry: metrics.NewRegistry(),
		ApiClient:       apiClient,
		DB:              db,
		Indexer:         ix,
		OpCfg:           &opCfg,
		OpSys:           opSys,
		L1Client:        opSys.Clients["l1"],
		L2Client:        opSys.Clients["sequencer"],
	}
}

func setupTestDatabase(t *testing.T) string {
	user := os.Getenv("DB_USER")
	require.NotEmpty(t, user, "DB_USER env variable expected to instantiate test database")

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

	dbConfig := config.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Name:     dbName,
		User:     user,
		Password: "",
	}

	noopLog := log.NewLogger(log.DiscardHandler())
	db, err := database.NewDB(context.Background(), noopLog, dbConfig)
	require.NoError(t, err)
	defer db.Close()

	err = db.ExecuteSQLMigration("../migrations")
	require.NoError(t, err)

	t.Logf("database %s setup and migrations executed", dbName)
	return dbName
}
