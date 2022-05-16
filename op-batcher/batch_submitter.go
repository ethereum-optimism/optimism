package op_batcher

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
	"github.com/ethereum-optimism/optimism/op-batcher/sequencer"
	proposer "github.com/ethereum-optimism/optimism/op-proposer"
	"github.com/ethereum-optimism/optimism/op-proposer/rollupclient"
	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/urfave/cli"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

// Main is the entrypoint into the Batch Submitter. This method returns a
// closure that executes the service and blocks until the service exits. The use
// of a closure allows the parameters bound to the top-level main package, e.g.
// GitVersion, to be captured and used once the function is executed.
func Main(version string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		cfg := NewConfig(ctx)

		// Set up our logging to stdout.
		var logHandler log.Handler
		if cfg.LogTerminal {
			logHandler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
		} else {
			logHandler = log.StreamHandler(os.Stdout, log.JSONFormat())
		}

		logLevel, err := log.LvlFromString(cfg.LogLevel)
		if err != nil {
			return err
		}

		l := log.New()
		l.SetHandler(log.LvlFilterHandler(logLevel, logHandler))

		l.Info("Initializing Batch Submitter")

		batchSubmitter, err := NewBatchSubmitter(cfg, version, l)
		if err != nil {
			l.Error("Unable to create Batch Submitter", "error", err)
			return err
		}

		l.Info("Starting Batch Submitter")

		if err := batchSubmitter.Start(); err != nil {
			l.Error("Unable to start Batch Submitter", "error", err)
			return err
		}
		defer batchSubmitter.Stop()

		l.Info("Batch Submitter started")

		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)
		<-interruptChannel

		return nil
	}
}

// BatchSubmitter encapsulates a service responsible for submitting L2 tx
// batches to L1 for availability.
type BatchSubmitter struct {
	ctx              context.Context
	sequencerService *proposer.Service
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitter(
	cfg Config,
	gitVersion string,
	l log.Logger,
) (*BatchSubmitter, error) {

	ctx := context.Background()

	// Parse wallet private key that will be used to submit L2 txs to the batch
	// inbox address.
	wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
	if err != nil {
		return nil, err
	}

	sequencerPrivKey, err := wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: cfg.SequencerHDPath,
		},
	})
	if err != nil {
		return nil, err
	}

	batchInboxAddress, err := parseAddress(cfg.SequencerBatchInboxAddress)
	if err != nil {
		return nil, err
	}

	genesisHash := common.HexToHash(cfg.SequencerGenesisHash)

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	l2Client, err := dialEthClientWithTimeout(ctx, cfg.L2EthRpc)
	if err != nil {
		return nil, err
	}

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	historyDB, err := db.OpenJSONFileDatabase(
		cfg.SequencerHistoryDBFilename, 600, genesisHash,
	)
	if err != nil {
		return nil, err
	}

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	txManagerConfig := txmgr.Config{
		Log:                       l,
		Name:                      "Batch Submitter",
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	sequencerDriver, err := sequencer.NewDriver(sequencer.Config{
		Log:               l,
		Name:              "Batch Submitter",
		L1Client:          l1Client,
		L2Client:          l2Client,
		RollupClient:      rollupClient,
		MinL1TxSize:       cfg.MinL1TxSize,
		MaxL1TxSize:       cfg.MaxL1TxSize,
		BatchInboxAddress: batchInboxAddress,
		HistoryDB:         historyDB,
		ChainID:           chainID,
		PrivKey:           sequencerPrivKey,
	})
	if err != nil {
		return nil, err
	}

	sequencerService := proposer.NewService(proposer.ServiceConfig{
		Log:             l,
		Context:         ctx,
		Driver:          sequencerDriver,
		PollInterval:    cfg.PollInterval,
		L1Client:        l1Client,
		TxManagerConfig: txManagerConfig,
	})

	return &BatchSubmitter{
		ctx:              ctx,
		sequencerService: sequencerService,
	}, nil
}

func (l *BatchSubmitter) Start() error {
	return l.sequencerService.Start()
}

func (l *BatchSubmitter) Stop() {
	_ = l.sequencerService.Stop()
}

// dialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func dialEthClientWithTimeout(ctx context.Context, url string) (
	*ethclient.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	return ethclient.DialContext(ctxt, url)
}

// dialRollupClientWithTimeout attempts to dial the RPC provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func dialRollupClientWithTimeout(ctx context.Context, url string) (*rollupclient.RollupClient, error) {
	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	client, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, err
	}

	return rollupclient.NewRollupClient(client), nil
}

// parseAddress parses an ETH address from a hex string. This method will fail if
// the address is not a valid hexadecimal address.
func parseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}
