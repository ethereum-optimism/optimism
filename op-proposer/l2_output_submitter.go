package op_proposer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/drivers/l2output"
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

// Main is the entrypoint into the L2 Output Submitter. This method returns a
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

		l.Info("Initializing L2 Output Submitter")

		l2OutputSubmitter, err := NewL2OutputSubmitter(cfg, version, l)
		if err != nil {
			l.Error("Unable to create L2 Output Submitter", "error", err)
			return err
		}

		l.Info("Starting L2 Output Submitter")

		if err := l2OutputSubmitter.Start(); err != nil {
			l.Error("Unable to start L2 Output Submitter", "error", err)
			return err
		}
		defer l2OutputSubmitter.Stop()

		l.Info("L2 Output Submitter started")

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

// L2OutputSubmitter encapsulates a service responsible for submitting
// L2Outputs to the L2OutputOracle contract.
type L2OutputSubmitter struct {
	ctx             context.Context
	l2OutputService *Service
}

// NewL2OutputSubmitter initializes the L2OutputSubmitter, gathering any resources
// that will be needed during operation.
func NewL2OutputSubmitter(
	cfg Config,
	gitVersion string,
	l log.Logger,
) (*L2OutputSubmitter, error) {

	ctx := context.Background()

	// Parse l2output wallet private key and L2OO contract address.
	wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
	if err != nil {
		return nil, err
	}

	l2OutputPrivKey, err := wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: cfg.L2OutputHDPath,
		},
	})
	if err != nil {
		return nil, err
	}

	l2ooAddress, err := parseAddress(cfg.L2OOAddress)
	if err != nil {
		return nil, err
	}

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

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	txManagerConfig := txmgr.Config{
		Log:                       l,
		Name:                      "L2Output Submitter",
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	l2OutputDriver, err := l2output.NewDriver(l2output.Config{
		Log:          l,
		Name:         "L2Output Submitter",
		L1Client:     l1Client,
		L2Client:     l2Client,
		RollupClient: rollupClient,
		L2OOAddr:     l2ooAddress,
		ChainID:      chainID,
		PrivKey:      l2OutputPrivKey,
	})
	if err != nil {
		return nil, err
	}

	l2OutputService := NewService(ServiceConfig{
		Log:             l,
		Context:         ctx,
		Driver:          l2OutputDriver,
		PollInterval:    cfg.PollInterval,
		L1Client:        l1Client,
		TxManagerConfig: txManagerConfig,
	})

	return &L2OutputSubmitter{
		ctx:             ctx,
		l2OutputService: l2OutputService,
	}, nil
}

func (l *L2OutputSubmitter) Start() error {
	return l.l2OutputService.Start()
}

func (l *L2OutputSubmitter) Stop() {
	_ = l.l2OutputService.Stop()
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
