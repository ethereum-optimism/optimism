package batchsubmitter

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/drivers/proposer"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/drivers/sequencer"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/txmgr"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/utils"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

// Main is the entrypoint into the batch submitter service. This method returns
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

		log.Info("Initializing batch submitter")

		batchSubmitter, err := NewBatchSubmitter(cfg, gitVersion)
		if err != nil {
			log.Error("Unable to create batch submitter", "error", err)
			return err
		}

		log.Info("Starting batch submitter")

		if err := batchSubmitter.Start(); err != nil {
			return err
		}
		defer batchSubmitter.Stop()

		log.Info("Batch submitter started")

		<-(chan struct{})(nil)

		return nil
	}
}

// BatchSubmitter is a service that configures the necessary resources for
// running the TxBatchSubmitter and StateBatchSubmitter sub-services.
type BatchSubmitter struct {
	ctx              context.Context
	cfg              Config
	l1Client         *ethclient.Client
	l2Client         *l2ethclient.Client
	sequencerPrivKey *ecdsa.PrivateKey
	proposerPrivKey  *ecdsa.PrivateKey
	ctcAddress       common.Address
	sccAddress       common.Address

	batchTxService    *Service
	batchStateService *Service
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed by the TxBatchSubmitter and StateBatchSubmitter
// sub-services.
func NewBatchSubmitter(cfg Config, gitVersion string) (*BatchSubmitter, error) {
	ctx := context.Background()

	// Set up our logging. If Sentry is enabled, we will use our custom
	// log handler that logs to stdout and forwards any error messages to
	// Sentry for collection. Otherwise, logs will only be posted to stdout.
	var logHandler log.Handler
	if cfg.SentryEnable {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDsn,
			Environment:      cfg.EthNetworkName,
			Release:          "batch-submitter@" + gitVersion,
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

	// Parse sequencer private key and CTC contract address.
	sequencerPrivKey, ctcAddress, err := parseWalletPrivKeyAndContractAddr(
		"Sequencer", cfg.Mnemonic, cfg.SequencerHDPath,
		cfg.SequencerPrivateKey, cfg.CTCAddress,
	)
	if err != nil {
		return nil, err
	}

	// Parse proposer private key and SCC contract address.
	proposerPrivKey, sccAddress, err := parseWalletPrivKeyAndContractAddr(
		"Proposer", cfg.Mnemonic, cfg.ProposerHDPath,
		cfg.ProposerPrivateKey, cfg.SCCAddress,
	)
	if err != nil {
		return nil, err
	}

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

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	txManagerConfig := txmgr.Config{
		MinGasPrice:          utils.GasPriceFromGwei(1),
		MaxGasPrice:          utils.GasPriceFromGwei(cfg.MaxGasPriceInGwei),
		GasRetryIncrement:    utils.GasPriceFromGwei(cfg.GasRetryIncrement),
		ResubmissionTimeout:  cfg.ResubmissionTimeout,
		ReceiptQueryInterval: time.Second,
		NumConfirmations:     cfg.NumConfirmations,
	}

	var batchTxService *Service
	if cfg.RunTxBatchSubmitter {
		batchTxDriver, err := sequencer.NewDriver(sequencer.Config{
			Name:        "Sequencer",
			L1Client:    l1Client,
			L2Client:    l2Client,
			BlockOffset: cfg.BlockOffset,
			MaxTxSize:   cfg.MaxL1TxSize,
			CTCAddr:     ctcAddress,
			ChainID:     chainID,
			PrivKey:     sequencerPrivKey,
		})
		if err != nil {
			return nil, err
		}

		batchTxService = NewService(ServiceConfig{
			Context:         ctx,
			Driver:          batchTxDriver,
			PollInterval:    cfg.PollInterval,
			ClearPendingTx:  cfg.ClearPendingTxs,
			L1Client:        l1Client,
			TxManagerConfig: txManagerConfig,
		})
	}

	var batchStateService *Service
	if cfg.RunStateBatchSubmitter {
		batchStateDriver, err := proposer.NewDriver(proposer.Config{
			Name:        "Proposer",
			L1Client:    l1Client,
			L2Client:    l2Client,
			BlockOffset: cfg.BlockOffset,
			MaxTxSize:   cfg.MaxL1TxSize,
			SCCAddr:     sccAddress,
			CTCAddr:     ctcAddress,
			ChainID:     chainID,
			PrivKey:     proposerPrivKey,
		})
		if err != nil {
			return nil, err
		}

		batchStateService = NewService(ServiceConfig{
			Context:         ctx,
			Driver:          batchStateDriver,
			PollInterval:    cfg.PollInterval,
			ClearPendingTx:  cfg.ClearPendingTxs,
			L1Client:        l1Client,
			TxManagerConfig: txManagerConfig,
		})
	}

	return &BatchSubmitter{
		ctx:               ctx,
		cfg:               cfg,
		l1Client:          l1Client,
		l2Client:          l2Client,
		sequencerPrivKey:  sequencerPrivKey,
		proposerPrivKey:   proposerPrivKey,
		ctcAddress:        ctcAddress,
		sccAddress:        sccAddress,
		batchTxService:    batchTxService,
		batchStateService: batchStateService,
	}, nil
}

func (b *BatchSubmitter) Start() error {
	if b.cfg.RunTxBatchSubmitter {
		if err := b.batchTxService.Start(); err != nil {
			return err
		}
	}
	if b.cfg.RunStateBatchSubmitter {
		if err := b.batchStateService.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (b *BatchSubmitter) Stop() {
	if b.cfg.RunTxBatchSubmitter {
		_ = b.batchTxService.Stop()
	}
	if b.cfg.RunStateBatchSubmitter {
		_ = b.batchStateService.Stop()
	}
}

// parseWalletPrivKeyAndContractAddr returns the wallet private key to use for
// sending transactions as well as the contract address to send to for a
// particular sub-service.
func parseWalletPrivKeyAndContractAddr(
	name string,
	mnemonic string,
	hdPath string,
	privKeyStr string,
	contractAddrStr string,
) (*ecdsa.PrivateKey, common.Address, error) {

	// Parse wallet private key from either privkey string or BIP39 mnemonic
	// and BIP32 HD derivation path.
	privKey, err := GetConfiguredPrivateKey(mnemonic, hdPath, privKeyStr)
	if err != nil {
		return nil, common.Address{}, err
	}

	// Parse the target contract address the wallet will send to.
	contractAddress, err := ParseAddress(contractAddrStr)
	if err != nil {
		return nil, common.Address{}, err
	}

	// Log wallet address rather than private key...
	walletAddress := crypto.PubkeyToAddress(privKey.PublicKey)

	log.Info(name+" wallet params parsed successfully", "wallet_address",
		walletAddress, "contract_address", contractAddress)

	return privKey, contractAddress, nil
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
