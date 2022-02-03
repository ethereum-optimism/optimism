package indexer

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	database "github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l1"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l2"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

var (
	// l2StartBlockNumber is the block number to start indexing l2 from
	l2StartBlockNumber = uint64(0)

	// l2StartBlockHash is the block hash to start indexing l2 from
	l2StartBlockHash = "0x7ca38a1916c42007829c55e69d3e9a73265554b586a499015373241b8a3fa48b"
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

		// The call to defer is done here so that any errors logged from
		// this point on are posted to Sentry before exiting.
		if cfg.SentryEnable {
			defer sentry.Flush(2 * time.Second)
		}

		log.Info("Initializing indexer")

		indexer, err := NewIndexer(cfg, gitVersion)
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
	ctx        context.Context
	cfg        Config
	l1Client   *ethclient.Client
	l2Client   *l2ethclient.Client
	ctcAddress common.Address
	sccAddress common.Address

	l1IndexingService *l1.Service
	l2IndexingService *l2.Service

	router *mux.Router
}

// NewIndexer initializes the Indexer, gathering any resources
// that will be needed by the TxIndexer and StateIndexer
// sub-services.
func NewIndexer(cfg Config, gitVersion string) (*Indexer, error) {
	ctx := context.Background()

	// Set up our logging. If Sentry is enabled, we will use our custom
	// log handler that logs to stdout and forwards any error messages to
	// Sentry for collection. Otherwise, logs will only be posted to stdout.
	var logHandler log.Handler
	if cfg.SentryEnable {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDsn,
			Environment:      cfg.EthNetworkName,
			Release:          "indexer@" + gitVersion,
			TracesSampleRate: traceRateToFloat64(cfg.SentryTraceRate),
			Debug:            false,
		})
		if err != nil {
			return nil, err
		}

		logHandler = SentryStreamHandler(os.Stdout, log.TerminalFormat(true))
	} else {
		logHandler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	}

	logLevel, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, err := dialL1EthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	l2Client, err := dialL2EthClientWithTimeout(ctx, cfg.L2EthRpc)
	if err != nil {
		return nil, err
	}

	if cfg.MetricsServerEnable {
		go runMetricsServer(cfg.MetricsHostname, cfg.MetricsPort)
	}

	db, err := database.NewDatabase(fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName))
	if err != nil {
		return nil, err
	}

	ctcAddress, err := ParseAddress(cfg.CTCAddress)
	if err != nil {
		return nil, err
	}

	l1IndexingService, err := l1.NewService(l1.ServiceConfig{
		Context:            ctx,
		L1Client:           l1Client,
		CTCAddr:            ctcAddress,
		DB:                 db,
		ConfDepth:          cfg.ConfDepth,
		MaxHeaderBatchSize: cfg.MaxHeaderBatchSize,
		StartBlockNumber:   cfg.StartBlockNumber,
		StartBlockHash:     cfg.StartBlockHash,
		Router:             mux.NewRouter(),
	})
	if err != nil {
		return nil, err
	}

	l2IndexingService, err := l2.NewService(l2.ServiceConfig{
		Context:            ctx,
		L2Client:           l2Client,
		DB:                 db,
		ConfDepth:          cfg.ConfDepth,
		MaxHeaderBatchSize: cfg.MaxHeaderBatchSize,
		StartBlockNumber:   l2StartBlockNumber,
		StartBlockHash:     l2StartBlockHash,
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
		router:            mux.NewRouter(),
	}, nil
}

func (b *Indexer) Serve(ctx context.Context) {
	b.router.HandleFunc("/v1/l1/status", b.l1IndexingService.GetIndexerStatus).Methods("GET")
	b.router.HandleFunc("/v1/l2/status", b.l2IndexingService.GetIndexerStatus).Methods("GET")
	b.router.HandleFunc("/v1/deposits/0x{address:[a-fA-F0-9]{40}}", b.l1IndexingService.GetDeposits).Methods("GET")
	b.router.HandleFunc("/v1/withdrawals/0x{address:[a-fA-F0-9]{40}}", b.l2IndexingService.GetWithdrawals).Methods("GET")

	http.ListenAndServe(":8080", b.router)
}

func (b *Indexer) Start() error {
	b.l1IndexingService.Start()
	b.l2IndexingService.Start()

	b.Serve(b.ctx)
	return nil
}

func (b *Indexer) Stop() {
	b.l1IndexingService.Stop()
	b.l2IndexingService.Stop()
}

// runMetricsServer spins up a prometheus metrics server at the provided
// hostname and port.
//
// NOTE: This method MUST be run as a goroutine.
func runMetricsServer(hostname string, port uint64) {
	metricsPortStr := strconv.FormatUint(port, 10)
	metricsAddr := fmt.Sprintf("%s:%s", hostname, metricsPortStr)

	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(metricsAddr, nil)
}

// dialL1EthClientWithTimeout attempts to dial the L1 provider using the
// provided URL. If the dial doesn't complete within defaultDialTimeout seconds,
// this method will return an error.
func dialL1EthClientWithTimeout(ctx context.Context, url string) (
	*ethclient.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	return ethclient.DialContext(ctxt, url)
}

// dialL2EthClientWithTimeout attempts to dial the L2 provider using the
// provided URL. If the dial doesn't complete within defaultDialTimeout seconds,
// this method will return an error.
func dialL2EthClientWithTimeout(ctx context.Context, url string) (
	*l2ethclient.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	return l2ethclient.DialContext(ctxt, url)
}

// traceRateToFloat64 converts a time.Duration into a valid float64 for the
// Sentry client. The client only accepts values between 0.0 and 1.0, so this
// method clamps anything greater than 1 second to 1.0.
func traceRateToFloat64(rate time.Duration) float64 {
	rate64 := float64(rate) / float64(time.Second)
	if rate64 > 1.0 {
		rate64 = 1.0
	}
	return rate64
}

func gasPriceFromGwei(gasPriceInGwei uint64) *big.Int {
	return new(big.Int).SetUint64(gasPriceInGwei * 1e9)
}
