package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
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

var supportedL2OutputVersion = eth.Bytes32{}

// Main is the entrypoint into the L2 Output Submitter. This method executes the
// service and blocks until the service exits.
func Main(version string, cliCtx *cli.Context) error {
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	l.Info("Initializing L2 Output Submitter")

	l2OutputSubmitter, err := NewL2OutputSubmitterFromCLIConfig(cfg, l)
	if err != nil {
		l.Error("Unable to create the L2 Output Submitter", "error", err)
		return err
	}

	l.Info("Starting L2 Output Submitter")
	ctx, cancel := context.WithCancel(context.Background())

	if err := l2OutputSubmitter.Start(); err != nil {
		cancel()
		l.Error("Unable to start L2 Output Submitter", "error", err)
		return err
	}
	defer l2OutputSubmitter.Stop()

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
		addr := l2OutputSubmitter.from
		opmetrics.LaunchBalanceMetrics(ctx, l, registry, "", l2OutputSubmitter.l1Client, addr)
	}

	rpcCfg := cfg.RPCConfig
	server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, version)
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

// L2OutputSubmitter is responsible for proposing outputs
type L2OutputSubmitter struct {
	txMgr txmgr.TxManager
	wg    sync.WaitGroup
	done  chan struct{}
	log   log.Logger

	ctx    context.Context
	cancel context.CancelFunc

	// L1Client is used to submit transactions to
	l1Client *ethclient.Client
	// RollupClient is used to retrieve output roots from
	rollupClient *sources.RollupClient

	l2ooContract    *bindings.L2OutputOracle
	rawL2ooContract *bind.BoundContract

	// AllowNonFinalized enables the proposal of safe, but non-finalized L2 blocks.
	// The L1 block-hash embedded in the proposal TX is checked and should ensure the proposal
	// is never valid on an alternative L1 chain that would produce different L2 data.
	// This option is not necessary when higher proposal latency is acceptable and L1 is healthy.
	allowNonFinalized bool
	// From is the address to send transactions from
	from common.Address
	// SignerFn is the function used to sign transactions
	signerFn opcrypto.SignerFn
	// How frequently to poll L2 for new finalized outputs
	pollInterval time.Duration
}

// NewL2OutputSubmitterFromCLIConfig creates a new L2 Output Submitter given the CLI Config
func NewL2OutputSubmitterFromCLIConfig(cfg CLIConfig, l log.Logger) (*L2OutputSubmitter, error) {
	signer, fromAddress, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, cfg.L2OutputHDPath, cfg.SignerConfig)
	if err != nil {
		return nil, err
	}

	l2ooAddress, err := parseAddress(cfg.L2OOAddress)
	if err != nil {
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the most expensive.
	ctx := context.Background()
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	txMgrConfg := txmgr.Config{
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
		From:                      fromAddress,
	}

	proposerCfg := Config{
		L2OutputOracleAddr: l2ooAddress,
		PollInterval:       cfg.PollInterval,
		TxManagerConfig:    txMgrConfg,
		L1Client:           l1Client,
		RollupClient:       rollupClient,
		AllowNonFinalized:  cfg.AllowNonFinalized,
		From:               fromAddress,
		SignerFnFactory:    signer,
	}

	return NewL2OutputSubmitter(proposerCfg, l)
}

// NewL2OutputSubmitter creates a new L2 Output Submitter
func NewL2OutputSubmitter(cfg Config, l log.Logger) (*L2OutputSubmitter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cCtx, cCancel := context.WithTimeout(ctx, defaultDialTimeout)
	chainID, err := cfg.L1Client.ChainID(cCtx)
	cCancel()
	if err != nil {
		cancel()
		return nil, err
	}
	signer := cfg.SignerFnFactory(chainID)
	cfg.TxManagerConfig.Signer = signer

	l2ooContract, err := bindings.NewL2OutputOracle(cfg.L2OutputOracleAddr, cfg.L1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	parsed, err := abi.JSON(strings.NewReader(bindings.L2OutputOracleMetaData.ABI))
	if err != nil {
		cancel()
		return nil, err
	}
	rawL2ooContract := bind.NewBoundContract(cfg.L2OutputOracleAddr, parsed, cfg.L1Client, cfg.L1Client, cfg.L1Client)

	return &L2OutputSubmitter{
		txMgr:  txmgr.NewSimpleTxManager("proposer", l, cfg.TxManagerConfig, cfg.L1Client),
		done:   make(chan struct{}),
		log:    l,
		ctx:    ctx,
		cancel: cancel,

		l1Client:     cfg.L1Client,
		rollupClient: cfg.RollupClient,

		l2ooContract:    l2ooContract,
		rawL2ooContract: rawL2ooContract,

		allowNonFinalized: cfg.AllowNonFinalized,
		from:              cfg.From,
		signerFn:          signer,
		pollInterval:      cfg.PollInterval,
	}, nil
}

func (l *L2OutputSubmitter) Start() error {
	l.wg.Add(1)
	go l.loop()
	return nil
}

func (l *L2OutputSubmitter) Stop() {
	l.cancel()
	close(l.done)
	l.wg.Wait()
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (l *L2OutputSubmitter) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	opts := &bind.TransactOpts{
		From: l.from,
		Signer: func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return l.signerFn(ctx, addr, tx)
		},
		Context: ctx,
		Nonce:   new(big.Int).SetUint64(tx.Nonce()),
		NoSend:  true,
	}
	return l.rawL2ooContract.RawTransact(opts, tx.Data())
}

// FetchNextOutputInfo gets the block number of the next proposal.
// It returns: the next block number, if the proposal should be made, error
func (l *L2OutputSubmitter) FetchNextOutputInfo(ctx context.Context) (*eth.OutputResponse, bool, error) {
	callOpts := &bind.CallOpts{
		From:    l.from,
		Context: ctx,
	}
	nextCheckpointBlock, err := l.l2ooContract.NextBlockNumber(callOpts)
	if err != nil {
		l.log.Error("proposer unable to get next block number", "err", err)
		return nil, false, err
	}
	// Fetch the current L2 heads
	status, err := l.rollupClient.SyncStatus(ctx)
	if err != nil {
		l.log.Error("proposer unable to get sync status", "err", err)
		return nil, false, err
	}
	// Use either the finalized or safe head depending on the config. Finalized head is default & safer.
	var currentBlockNumber *big.Int
	if l.allowNonFinalized {
		currentBlockNumber = new(big.Int).SetUint64(status.SafeL2.Number)
	} else {
		currentBlockNumber = new(big.Int).SetUint64(status.FinalizedL2.Number)
	}
	// Ensure that we do not submit a block in the future
	if currentBlockNumber.Cmp(nextCheckpointBlock) < 0 {
		l.log.Info("proposer submission interval has not elapsed", "currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextCheckpointBlock)
		return nil, false, nil
	}

	output, err := l.rollupClient.OutputAtBlock(ctx, nextCheckpointBlock.Uint64())
	if err != nil {
		l.log.Error("failed to fetch output at block %d: %w", nextCheckpointBlock, err)
		return nil, false, err
	}
	if output.Version != supportedL2OutputVersion {
		l.log.Error("unsupported l2 output version: %s", output.Version)
		return nil, false, errors.New("unsupported l2 output version")
	}
	if output.BlockRef.Number != nextCheckpointBlock.Uint64() { // sanity check, e.g. in case of bad RPC caching
		l.log.Error("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", nextCheckpointBlock, output.BlockRef.Number)
		return nil, false, errors.New("invalid blockNumber")
	}

	// Always propose if it's part of the Finalized L2 chain. Or if allowed, if it's part of the safe L2 chain.
	if !(output.BlockRef.Number <= output.Status.FinalizedL2.Number || (l.allowNonFinalized && output.BlockRef.Number <= output.Status.SafeL2.Number)) {
		l.log.Debug("not proposing yet, L2 block is not ready for proposal",
			"l2_proposal", output.BlockRef,
			"l2_safe", output.Status.SafeL2,
			"l2_finalized", output.Status.FinalizedL2,
			"allow_non_finalized", l.allowNonFinalized)
		return nil, false, nil
	}
	return output, true, nil
}

// CreateProposalTx transforms an output response into a signed output transaction.
// It does not send the transaction to the transaction pool.
func (l *L2OutputSubmitter) CreateProposalTx(ctx context.Context, output *eth.OutputResponse) (*types.Transaction, error) {
	nonce, err := l.l1Client.NonceAt(ctx, l.from, nil)
	if err != nil {
		l.log.Error("Failed to get nonce", "err", err, "from", l.from)
		return nil, err
	}

	opts := &bind.TransactOpts{
		From: l.from,
		Signer: func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return l.signerFn(ctx, addr, tx)
		},
		Context: ctx,
		Nonce:   new(big.Int).SetUint64(nonce),
		NoSend:  true,
	}

	tx, err := l.l2ooContract.ProposeL2Output(
		opts,
		output.OutputRoot,
		new(big.Int).SetUint64(output.BlockRef.Number),
		output.Status.CurrentL1.Hash,
		new(big.Int).SetUint64(output.Status.CurrentL1.Number))
	if err != nil {
		l.log.Error("failed to create the ProposeL2Output transaction", "err", err)
		return nil, err
	}
	return tx, nil
}

// SendTransaction sends a transaction through the transaction manager which handles automatic
// price bumping.
// It also hardcodes a timeout of 100s.
func (l *L2OutputSubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	// Wait until one of our submitted transactions confirms. If no
	// receipt is received it's likely our gas price was too low.
	cCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	l.log.Info("Sending transaction", "tx_hash", tx.Hash())
	receipt, err := l.txMgr.Send(cCtx, tx)
	if err != nil {
		l.log.Error("proposer unable to publish tx", "err", err)
		return err
	}

	// The transaction was successfully submitted
	l.log.Info("proposer tx successfully published", "tx_hash", receipt.TxHash)
	return nil
}

// loop is responsible for creating & submitting the next outputs
func (l *L2OutputSubmitter) loop() {
	defer l.wg.Done()

	ctx := l.ctx

	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
			output, shouldPropose, err := l.FetchNextOutputInfo(cCtx)
			if err != nil {
				l.log.Error("Failed to fetch next output", "err", err)
				cancel()
				break
			}
			if !shouldPropose {
				cancel()
				break
			}

			tx, err := l.CreateProposalTx(cCtx, output)
			if err != nil {
				l.log.Error("Failed to create proposal transaction", "err", err)
				cancel()
				break
			}
			if err := l.SendTransaction(cCtx, tx); err != nil {
				l.log.Error("Failed to send proposal transaction", "err", err)
				cancel()
				break
			}
			cancel()

		case <-l.done:
			return
		}
	}
}
