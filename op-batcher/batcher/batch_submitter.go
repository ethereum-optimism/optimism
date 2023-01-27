package batcher

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
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
func Main(version string, cliCtx *cli.Context) error {
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	l.Info("Initializing Batch Submitter")

	batchSubmitter, err := NewBatchSubmitterFromCLIConfig(cfg, l)
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

	ctx, cancel := context.WithCancel(context.Background())

	l.Info("Batch Submitter started")
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
		opmetrics.LaunchBalanceMetrics(ctx, l, registry, "", batchSubmitter.L1Client, batchSubmitter.From)
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
	_ = server.Stop()
	return nil

}

// BatchSubmitter encapsulates a service responsible for submitting L2 tx
// batches to L1 for availability.
type BatchSubmitter struct {
	Config // directly embed the config + sources

	txMgr *TransactionManager
	wg    sync.WaitGroup
	done  chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	// lastStoredBlock is the last block loaded into `state`. If it is empty it should be set to the l2 safe head.
	lastStoredBlock eth.BlockID

	state *channelManager
}

// NewBatchSubmitterFromCLIConfig initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitterFromCLIConfig(cfg CLIConfig, l log.Logger) (*BatchSubmitter, error) {
	ctx := context.Background()

	signer, fromAddress, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, cfg.SequencerHDPath, cfg.SignerConfig)
	if err != nil {
		return nil, err
	}

	batchInboxAddress, err := parseAddress(cfg.SequencerBatchInboxAddress)
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
		Name:                      "Batch Submitter",
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	batcherCfg := Config{
		L1Client:          l1Client,
		L2Client:          l2Client,
		RollupNode:        rollupClient,
		ChainID:           chainID,
		PollInterval:      cfg.PollInterval,
		TxManagerConfig:   txManagerConfig,
		From:              fromAddress,
		SignerFnFactory:   signer,
		BatchInboxAddress: batchInboxAddress,
		Channel: ChannelConfig{
			ChannelTimeout:   cfg.ChannelTimeout,
			MaxFrameSize:     cfg.MaxL1TxSize - 1,    // subtract 1 byte for version
			TargetFrameSize:  cfg.TargetL1TxSize - 1, // subtract 1 byte for version
			TargetNumFrames:  cfg.TargetNumFrames,
			ApproxComprRatio: cfg.ApproxComprRatio,
		},
	}

	return NewBatchSubmitter(batcherCfg, l)
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitter(cfg Config, l log.Logger) (*BatchSubmitter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	balance, err := cfg.L1Client.BalanceAt(ctx, cfg.From, nil)
	if err != nil {
		cancel()
		return nil, err
	}

	cfg.log = l
	cfg.log.Info("creating batch submitter", "submitter_addr", cfg.From, "submitter_bal", balance)

	return &BatchSubmitter{
		Config: cfg,
		txMgr:  NewTransactionManager(l, cfg.TxManagerConfig, cfg.BatchInboxAddress, cfg.ChainID, cfg.From, cfg.L1Client, cfg.SignerFnFactory(cfg.ChainID)),
		done:   make(chan struct{}),
		// TODO: this context only exists because the even loop doesn't reach done
		// if the tx manager is blocking forever due to e.g. insufficient balance.
		ctx:    ctx,
		cancel: cancel,
		state:  NewChannelManager(l, cfg.Channel),
	}, nil

}
