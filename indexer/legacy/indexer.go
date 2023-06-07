package legacy

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/services"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/indexer/metrics"
	"github.com/ethereum-optimism/optimism/indexer/server"
	"github.com/rs/cors"

	database "github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services/l1"
	"github.com/ethereum-optimism/optimism/indexer/services/l2"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

// Main is the entrypoint into the indexer service. This method returns
// a closure that executes the service and blocks until the service exits. The
// use of a closure allows the parameters bound to the top-level main package,
// e.g. GitVersion, to be captured and used once the function is executed.
func Main(gitVersion string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		cfg, err := NewConfig(ctx)
		if err != nil {
			return err
		}

		log.Info("Initializing indexer")

		indexer, err := NewIndexer(cfg)
		if err != nil {
			log.Error("Unable to create indexer", "error", err)
			return err
		}

		log.Info("Starting indexer")

		if err := indexer.Start(); err != nil {
			return err
		}
		defer indexer.Stop()

		log.Info("Indexer started")

		<-(chan struct{})(nil)

		return nil
	}
}

// Indexer is a service that configures the necessary resources for
// running the Sync and BlockHandler sub-services.
type Indexer struct {
	ctx      context.Context
	cfg      Config
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	l1IndexingService *l1.Service
	l2IndexingService *l2.Service
	airdropService    *services.Airdrop

	router  *mux.Router
	metrics *metrics.Metrics
	db      *database.Database
	server  *http.Server
}

// NewIndexer initializes the Indexer, gathering any resources
// that will be needed by the TxIndexer and StateIndexer
// sub-services.
func NewIndexer(cfg Config) (*Indexer, error) {
	ctx := context.Background()

	var logHandler log.Handler
	if cfg.LogTerminal {
		logHandler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	} else {
		logHandler = log.StreamHandler(os.Stdout, log.JSONFormat())
	}

	logLevel, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, rawl1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	l2Client, l2RPC, err := dialEthClientWithTimeout(ctx, cfg.L2EthRpc)
	if err != nil {
		return nil, err
	}

	m := metrics.NewMetrics(nil)

	if cfg.MetricsServerEnable {
		go func() {
			_, err := m.Serve(cfg.MetricsHostname, cfg.MetricsPort)
			if err != nil {
				log.Error("metrics server failed to start", "err", err)
			}
		}()
		log.Info("metrics server enabled", "host", cfg.MetricsHostname, "port", cfg.MetricsPort)
	}

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBName)
	if cfg.DBUser != "" {
		dsn += fmt.Sprintf(" user=%s", cfg.DBUser)
	}
	if cfg.DBPassword != "" {
		dsn += fmt.Sprintf(" password=%s", cfg.DBPassword)
	}
	db, err := database.NewDatabase(dsn)
	if err != nil {
		return nil, err
	}

	var addrManager services.AddressManager
	if cfg.Bedrock {
		addrManager, err = services.NewBedrockAddresses(
			l1Client,
			cfg.BedrockL1StandardBridgeAddress,
			cfg.BedrockOptimismPortalAddress,
		)
	} else {
		addrManager, err = services.NewLegacyAddresses(l1Client, common.HexToAddress(cfg.L1AddressManagerAddress))
	}
	if err != nil {
		return nil, err
	}

	l1IndexingService, err := l1.NewService(l1.ServiceConfig{
		Context:            ctx,
		Metrics:            m,
		L1Client:           l1Client,
		RawL1Client:        rawl1Client,
		ChainID:            new(big.Int).SetUint64(cfg.ChainID),
		AddressManager:     addrManager,
		DB:                 db,
		ConfDepth:          cfg.L1ConfDepth,
		MaxHeaderBatchSize: cfg.MaxHeaderBatchSize,
		StartBlockNumber:   cfg.L1StartBlockNumber,
		Bedrock:            cfg.Bedrock,
	})
	if err != nil {
		return nil, err
	}

	l2IndexingService, err := l2.NewService(l2.ServiceConfig{
		Context:            ctx,
		Metrics:            m,
		L2RPC:              l2RPC,
		L2Client:           l2Client,
		DB:                 db,
		ConfDepth:          cfg.L2ConfDepth,
		MaxHeaderBatchSize: cfg.MaxHeaderBatchSize,
		StartBlockNumber:   uint64(0),
		Bedrock:            cfg.Bedrock,
	})
	if err != nil {
		return nil, err
	}

	return &Indexer{
		ctx:               ctx,
		cfg:               cfg,
		l1Client:          l1Client,
		l2Client:          l2Client,
		l1IndexingService: l1IndexingService,
		l2IndexingService: l2IndexingService,
		airdropService:    services.NewAirdrop(db, m),
		router:            mux.NewRouter(),
		metrics:           m,
		db:                db,
	}, nil
}

// Serve spins up a REST API server at the given hostname and port.
func (b *Indexer) Serve() error {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})

	b.router.HandleFunc("/v1/l1/status", b.l1IndexingService.GetIndexerStatus).Methods("GET")
	b.router.HandleFunc("/v1/l2/status", b.l2IndexingService.GetIndexerStatus).Methods("GET")
	b.router.HandleFunc("/v1/deposits/0x{address:[a-fA-F0-9]{40}}", b.l1IndexingService.GetDeposits).Methods("GET")
	b.router.HandleFunc("/v1/withdrawal/0x{hash:[a-fA-F0-9]{64}}", b.l2IndexingService.GetWithdrawalBatch).Methods("GET")
	b.router.HandleFunc("/v1/withdrawals/0x{address:[a-fA-F0-9]{40}}", b.l2IndexingService.GetWithdrawals).Methods("GET")
	b.router.HandleFunc("/v1/airdrops/0x{address:[a-fA-F0-9]{40}}", b.airdropService.GetAirdrop)
	b.router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error("Error handling /healthz", "error", err)
		}
	})

	middleware := server.LoggingMiddleware(b.metrics, log.New("service", "server"))

	port := strconv.FormatUint(b.cfg.RESTPort, 10)
	addr := net.JoinHostPort(b.cfg.RESTHostname, port)

	b.server = &http.Server{
		Addr:    addr,
		Handler: middleware(c.Handler(b.router)),
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- b.server.ListenAndServe()
	}()

	// Capture server startup errors
	<-time.After(10 * time.Millisecond)

	select {
	case err := <-errCh:
		return err
	default:
		log.Info("indexer REST server listening on", "addr", addr)
		return nil
	}
}

// Start starts the starts the indexing service on L1 and L2 chains and also
// starts the REST server.
func (b *Indexer) Start() error {
	if b.cfg.DisableIndexer {
		log.Info("indexer disabled, only serving data")
	} else {
		err := b.l1IndexingService.Start()
		if err != nil {
			return err
		}
		err = b.l2IndexingService.Start()
		if err != nil {
			return err
		}
	}

	return b.Serve()
}

// Stop stops the indexing service on L1 and L2 chains.
func (b *Indexer) Stop() {
	b.db.Close()

	if b.server != nil {
		// background context here so it waits for
		// conns to close
		_ = b.server.Shutdown(context.Background())
	}

	if !b.cfg.DisableIndexer {
		b.l1IndexingService.Stop()
		b.l2IndexingService.Stop()
	}
}

// dialL1EthClientWithTimeout attempts to dial the L1 provider using the
// provided URL. If the dial doesn't complete within defaultDialTimeout seconds,
// this method will return an error.
func dialEthClientWithTimeout(ctx context.Context, url string) (
	*ethclient.Client, *rpc.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	c, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, nil, err
	}
	return ethclient.NewClient(c), c, nil
}
