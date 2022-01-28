package l2os

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/l2os/txmgr"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
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

		log.Info("Initializing L2 Output Submitter")

		l2OutputSubmitter, err := NewL2OutputSubmitter(cfg, version)
		if err != nil {
			log.Error("Unable to create L2 Output Submitter", "error", err)
			return err
		}

		log.Info("Starting L2 Output Submitter")

		if err := l2OutputSubmitter.Start(); err != nil {
			log.Error("Unable to start L2 Output Submitter", "error", err)
			return err
		}
		defer l2OutputSubmitter.Stop()

		log.Info("L2 Output Submitter started")

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
	ctx context.Context
}

// NewL2OutputSubmitter initializes the L2OutputSubmitter, gathering any resources
// that will be needed during operation.
func NewL2OutputSubmitter(cfg Config, gitVersion string) (*L2OutputSubmitter, error) {
	ctx := context.Background()

	// Set up our logging to stdout.
	logHandler := log.StreamHandler(os.Stdout, log.TerminalFormat(true))

	logLevel, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

	// Parse l2output wallet private key and L2OO contract address.
	wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
	if err != nil {
		return nil, err
	}

	_, err = wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: cfg.L2OutputHDPath,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = parseAddress(cfg.L2OOAddress)
	if err != nil {
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	_, err = dialEthClientWithTimeout(ctx, cfg.L2EthRpc)
	if err != nil {
		return nil, err
	}

	_, err = l1Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	_ = txmgr.Config{
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	return &L2OutputSubmitter{
		ctx: ctx,
	}, nil
}

func (l *L2OutputSubmitter) Start() error {
	return nil
}

func (l *L2OutputSubmitter) Stop() {
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

// parseAddress parses an ETH addres from a hex string. This method will fail if
// the address is not a valid hexidecimal address.
func parseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}
