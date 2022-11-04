package op_proposer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/drivers/l2output"
	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
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
	return func(cliCtx *cli.Context) error {
		cfg := NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(cfg.LogConfig)
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

		ctx, cancel := context.WithCancel(context.Background())

		l.Info("L2 Output Submitter started")
		pprofConfig := cfg.PprofConfig
		if pprofConfig.Enabled {
			l.Info("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
			go func() {
				if err := oppprof.ListenAndServe(ctx, pprofConfig.ListenAddr, pprofConfig.ListenPort); err != nil {
					l.Error("error starting pprof", "err", err)
				}
			}()
		}

		registry := opmetrics.NewRegistry()
		metricsCfg := cfg.MetricsConfig
		if metricsCfg.Enabled {
			l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
			go func() {
				if err := opmetrics.ListenAndServe(ctx, registry, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
					l.Error("error starting metrics server", err)
				}
			}()
			addr := l2OutputSubmitter.l2OutputService.cfg.Driver.WalletAddr()
			opmetrics.LaunchBalanceMetrics(ctx, l, registry, "", l2OutputSubmitter.l2OutputService.cfg.L1Client, addr)
		}

		rpcCfg := cfg.RPCConfig
		server := oprpc.NewServer(
			rpcCfg.ListenAddr,
			rpcCfg.ListenPort,
			version,
		)
		if err := server.Start(); err != nil {
			cancel()
			return fmt.Errorf("error starting RPC server: %w", err)
		}

		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)
		<-interruptChannel
		cancel()

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
	var l2OutputPrivKey *ecdsa.PrivateKey
	var err error

	if cfg.PrivateKey != "" && cfg.Mnemonic != "" {
		return nil, errors.New("cannot specify both a private key and a mnemonic")
	}

	if cfg.PrivateKey == "" {
		// Parse l2output wallet private key and L2OO contract address.
		wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
		if err != nil {
			return nil, err
		}

		l2OutputPrivKey, err = wallet.PrivateKey(accounts.Account{
			URL: accounts.URL{
				Path: cfg.L2OutputHDPath,
			},
		})
		if err != nil {
			return nil, err
		}
	} else {
		l2OutputPrivKey, err = crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
		if err != nil {
			return nil, err
		}
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
		Log:               l,
		Name:              "L2Output Submitter",
		L1Client:          l1Client,
		RollupClient:      rollupClient,
		AllowNonFinalized: cfg.AllowNonFinalized,
		L2OOAddr:          l2ooAddress,
		ChainID:           chainID,
		PrivKey:           l2OutputPrivKey,
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
func dialRollupClientWithTimeout(ctx context.Context, url string) (*sources.RollupClient, error) {
	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	rpcCl, err := rpc.DialContext(ctxt, url)
	if err != nil {
		return nil, err
	}

	return sources.NewRollupClient(client.NewBaseRPCClient(rpcCl)), nil
}

// parseAddress parses an ETH address from a hex string. This method will fail if
// the address is not a valid hexadecimal address.
func parseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}
