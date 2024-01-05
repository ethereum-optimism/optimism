package indexer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/etl"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processors"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

// Indexer contains the necessary resources for
// indexing the configured L1 and L2 chains
type Indexer struct {
	log log.Logger
	DB  *database.DB

	l1Client node.EthClient
	l2Client node.EthClient

	// api server only really serves a /health endpoint here, but this may change in the future
	apiServer *httputil.HTTPServer

	metricsServer *httputil.HTTPServer

	metricsRegistry *prometheus.Registry

	L1ETL           *etl.L1ETL
	L2ETL           *etl.L2ETL
	BridgeProcessor *processors.BridgeProcessor

	// shutdown requests the service that maintains the indexer to shut down,
	// and provides the error-cause of the critical failure (if any).
	shutdown context.CancelCauseFunc

	stopped atomic.Bool
}

// NewIndexer initializes an instance of the Indexer
func NewIndexer(ctx context.Context, log log.Logger, cfg *config.Config, shutdown context.CancelCauseFunc) (*Indexer, error) {
	out := &Indexer{
		log:             log,
		metricsRegistry: metrics.NewRegistry(),
		shutdown:        shutdown,
	}
	if err := out.initFromConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, out.Stop(ctx))
	}
	return out, nil
}

func (ix *Indexer) Start(ctx context.Context) error {
	// If any of these services has a critical failure,
	// the service can request a shutdown, while providing the error cause.
	if err := ix.L1ETL.Start(); err != nil {
		return fmt.Errorf("failed to start L1 ETL: %w", err)
	}
	if err := ix.L2ETL.Start(); err != nil {
		return fmt.Errorf("failed to start L2 ETL: %w", err)
	}
	if err := ix.BridgeProcessor.Start(); err != nil {
		return fmt.Errorf("failed to start bridge processor: %w", err)
	}
	return nil
}

func (ix *Indexer) Stop(ctx context.Context) error {
	var result error

	if ix.L1ETL != nil {
		if err := ix.L1ETL.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close L1 ETL: %w", err))
		}
	}

	if ix.L2ETL != nil {
		if err := ix.L2ETL.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close L2 ETL: %w", err))
		}
	}

	if ix.BridgeProcessor != nil {
		if err := ix.BridgeProcessor.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close bridge processor: %w", err))
		}
	}

	// Now that the ETLs are closed, we can stop the RPC clients
	if ix.l1Client != nil {
		ix.l1Client.Close()
	}
	if ix.l2Client != nil {
		ix.l2Client.Close()
	}

	if ix.apiServer != nil {
		if err := ix.apiServer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close indexer API server: %w", err))
		}
	}

	// DB connection can be closed last, after all its potential users have shut down
	if ix.DB != nil {
		if err := ix.DB.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close DB: %w", err))
		}
	}

	if ix.metricsServer != nil {
		if err := ix.metricsServer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close metrics server: %w", err))
		}
	}

	ix.stopped.Store(true)

	ix.log.Info("indexer stopped")

	return result
}

func (ix *Indexer) Stopped() bool {
	return ix.stopped.Load()
}

func (ix *Indexer) initFromConfig(ctx context.Context, cfg *config.Config) error {
	if err := ix.initRPCClients(ctx, cfg.RPCs); err != nil {
		return fmt.Errorf("failed to start RPC clients: %w", err)
	}
	if err := ix.initDB(ctx, cfg.DB); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}
	if err := ix.initL1ETL(cfg.Chain); err != nil {
		return fmt.Errorf("failed to init L1 ETL: %w", err)
	}
	if err := ix.initL2ETL(cfg.Chain); err != nil {
		return fmt.Errorf("failed to init L2 ETL: %w", err)
	}
	if err := ix.initBridgeProcessor(cfg.Chain); err != nil {
		return fmt.Errorf("failed to init Bridge-Processor: %w", err)
	}
	if err := ix.startHttpServer(ctx, cfg.HTTPServer); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	if err := ix.startMetricsServer(ctx, cfg.MetricsServer); err != nil {
		return fmt.Errorf("failed to start Metrics server: %w", err)
	}
	return nil
}

func (ix *Indexer) initRPCClients(ctx context.Context, rpcsConfig config.RPCsConfig) error {
	l1EthClient, err := node.DialEthClient(ctx, rpcsConfig.L1RPC, node.NewMetrics(ix.metricsRegistry, "l1"))
	if err != nil {
		return fmt.Errorf("failed to dial L1 client: %w", err)
	}
	ix.l1Client = l1EthClient

	l2EthClient, err := node.DialEthClient(ctx, rpcsConfig.L2RPC, node.NewMetrics(ix.metricsRegistry, "l2"))
	if err != nil {
		return fmt.Errorf("failed to dial L2 client: %w", err)
	}
	ix.l2Client = l2EthClient
	return nil
}

func (ix *Indexer) initDB(ctx context.Context, cfg config.DBConfig) error {
	db, err := database.NewDB(ctx, ix.log, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	ix.DB = db
	return nil
}

func (ix *Indexer) initL1ETL(chainConfig config.ChainConfig) error {
	l1Cfg := etl.Config{
		LoopIntervalMsec:  chainConfig.L1PollingInterval,
		HeaderBufferSize:  chainConfig.L1HeaderBufferSize,
		ConfirmationDepth: big.NewInt(int64(chainConfig.L1ConfirmationDepth)),
		StartHeight:       big.NewInt(int64(chainConfig.L1StartingHeight)),
	}
	l1Etl, err := etl.NewL1ETL(l1Cfg, ix.log, ix.DB, etl.NewMetrics(ix.metricsRegistry, "l1"),
		ix.l1Client, chainConfig.L1Contracts, ix.shutdown)
	if err != nil {
		return err
	}
	ix.L1ETL = l1Etl
	return nil
}

func (ix *Indexer) initL2ETL(chainConfig config.ChainConfig) error {
	// L2 (defaults to predeploy contracts)
	l2Cfg := etl.Config{
		LoopIntervalMsec:  chainConfig.L2PollingInterval,
		HeaderBufferSize:  chainConfig.L2HeaderBufferSize,
		ConfirmationDepth: big.NewInt(int64(chainConfig.L2ConfirmationDepth)),
	}
	l2Etl, err := etl.NewL2ETL(l2Cfg, ix.log, ix.DB, etl.NewMetrics(ix.metricsRegistry, "l2"),
		ix.l2Client, chainConfig.L2Contracts, ix.shutdown)
	if err != nil {
		return err
	}
	ix.L2ETL = l2Etl
	return nil
}

func (ix *Indexer) initBridgeProcessor(chainConfig config.ChainConfig) error {
	bridgeProcessor, err := processors.NewBridgeProcessor(
		ix.log, ix.DB, bridge.NewMetrics(ix.metricsRegistry), ix.L1ETL, ix.L2ETL, chainConfig, ix.shutdown)
	if err != nil {
		return err
	}
	ix.BridgeProcessor = bridgeProcessor
	return nil
}

func (ix *Indexer) startHttpServer(ctx context.Context, cfg config.ServerConfig) error {
	ix.log.Debug("starting http server...", "port", cfg.Port)

	r := chi.NewRouter()
	r.Use(middleware.Heartbeat("/healthz"))

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	srv, err := httputil.StartHTTPServer(addr, r)
	if err != nil {
		return fmt.Errorf("http server failed to start: %w", err)
	}
	ix.apiServer = srv
	ix.log.Info("http server started", "addr", srv.Addr())
	return nil
}

func (ix *Indexer) startMetricsServer(ctx context.Context, cfg config.ServerConfig) error {
	ix.log.Debug("starting metrics server...", "port", cfg.Port)
	srv, err := metrics.StartServer(ix.metricsRegistry, cfg.Host, cfg.Port)
	if err != nil {
		return fmt.Errorf("metrics server failed to start: %w", err)
	}
	ix.metricsServer = srv
	ix.log.Info("metrics server started", "addr", srv.Addr())
	return nil
}
