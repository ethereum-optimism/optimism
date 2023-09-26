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
	"github.com/ethereum-optimism/optimism/indexer/client"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

type E2ETestSuite struct {
	t *testing.T

	// API
	Client *client.Client
	API    *api.API

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

func createIndexerTestSuite(t *testing.T) E2ETestSuite {
	return buildE2ETestSuite(t, false)
}

func createAPITestSuite(t *testing.T) E2ETestSuite {
	return buildE2ETestSuite(t, true)
}

func buildE2ETestSuite(t *testing.T, withAPI bool) E2ETestSuite {
	dbUser := os.Getenv("DB_USER")
	dbName := setupTestDatabase(t)

	// Discard the Global Logger as each component
	// has its own configured logger
	log.Root().SetHandler(log.DiscardHandler())

	// Rollup System Configuration and Start
	opCfg := op_e2e.DefaultSystemConfig(t)
	opCfg.DeployConfig.FinalizationPeriodSeconds = 2
	opSys, err := opCfg.Start(t)
	require.NoError(t, err)
	t.Cleanup(func() { opSys.Close() })

	if !withAPI {
		// E2E tests can run on the order of magnitude of minutes. Once
		// the system is running, mark this test for Parallel execution
		// E2E API tests will not be marked for Parallel execution as they
		// will try consuming the same port and fail. This can be fixed if a
		// unique sport is dynamically allocated for each test.
		t.Parallel()

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
			},
		},
		HTTPServer:    config.ServerConfig{Host: "127.0.0.1", Port: 0},
		MetricsServer: config.ServerConfig{Host: "127.0.0.1", Port: 0},
	}

	db, err := database.NewDB(indexerCfg.DB)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	indexerLog := testlog.Logger(t, log.LvlInfo).New("role", "indexer")
	indexer, err := indexer.NewIndexer(indexerLog, db, indexerCfg.Chain, indexerCfg.RPCs, indexerCfg.HTTPServer, indexerCfg.MetricsServer)
	require.NoError(t, err)

	appCtx, appStop := context.WithCancel(context.Background())
	go func() {
		err := indexer.Run(appCtx)
		if err != nil { // panicking here ensures that the test will exit
			// during service failure. Using t.Fail() wouldn't be caught
			// until all awaiting routines finish which would never happen.
			panic(err)
		}
	}()

	var indexerAPI *api.API
	var indexerClient *client.Client

	if withAPI {
		apiLog := testlog.Logger(t, log.LvlInfo).New("role", "indexer_api")

		apiCfg := config.ServerConfig{
			Host: "127.0.0.1",
			Port: 4321,
		}

		mCfg := config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
		}

		indexerAPI = api.NewApi(apiLog, db.BridgeTransfers, apiCfg, mCfg)
		go func() {
			err := indexerAPI.Start(appCtx)
			if err != nil {
				panic(err)
			}
		}()

		indexerClient, err = client.NewClient(&client.Config{
			PaginationLimit: 100,
			BaseURL:         fmt.Sprintf("http://%s:%d", indexerCfg.HTTPServer.Host, indexerCfg.HTTPServer.Port),
		})
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		appStop()
	})

	return E2ETestSuite{
		t:        t,
		API:      indexerAPI,
		Client:   indexerClient,
		DB:       db,
		Indexer:  indexer,
		OpCfg:    &opCfg,
		OpSys:    opSys,
		L1Client: opSys.Clients["l1"],
		L2Client: opSys.Clients["sequencer"],
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
	// NewDB will create the database schema
	db, err := database.NewDB(dbConfig)
	require.NoError(t, err)
	defer db.Close()
	err = db.ExecuteSQLMigration("../migrations")
	require.NoError(t, err)

	t.Logf("database %s setup and migrations executed", dbName)
	return dbName
}
