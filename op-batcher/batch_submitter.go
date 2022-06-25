package op_batcher

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
	"github.com/ethereum-optimism/optimism/op-batcher/sequencer"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

		batchSubmitter, err := NewBatchSubmitter(cfg, l)
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
	ctx   context.Context
	txMgr txmgr.TxManager
	cfg   sequencer.Config
	wg    sync.WaitGroup
	done  chan struct{}
	log   log.Logger

	l2HeadNumber uint64

	ch *derive.ChannelOut
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitter(cfg Config, l log.Logger) (*BatchSubmitter, error) {
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

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	l2Client, err := dialEthClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	historyDB, err := db.OpenJSONFileDatabase(cfg.SequencerHistoryDBFilename)
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

	batcherCfg := sequencer.Config{
		Log:                 l,
		Name:                "Batch Submitter",
		L1Client:            l1Client,
		L2Client:            l2Client,
		MinL1TxSize:         cfg.MinL1TxSize,
		MaxL1TxSize:         cfg.MaxL1TxSize,
		MaxBlocksPerChannel: cfg.MaxBlocksPerChannel,
		BatchInboxAddress:   batchInboxAddress,
		HistoryDB:           historyDB,
		ChannelTimeout:      cfg.ChannelTimeout,
		ChainID:             chainID,
		PrivKey:             sequencerPrivKey,
		PollInterval:        cfg.PollInterval,
	}

	return &BatchSubmitter{
		cfg:   batcherCfg,
		txMgr: txmgr.NewSimpleTxManager("batcher", txManagerConfig, l1Client),
		done:  make(chan struct{}),
		log:   l,
	}, nil
}

func (l *BatchSubmitter) Start() error {
	l.wg.Add(1)
	go l.loop()
	return nil
}

func (l *BatchSubmitter) Stop() {
	close(l.done)
	l.wg.Wait()
}

func (l *BatchSubmitter) loop() {
	ctx := context.Background()
	defer l.wg.Done()

	ticker := time.NewTicker(l.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Do the simplest thing of one channel per block
			// TODO: Do one channel per epoch
			head, err := l.cfg.L2Client.BlockByNumber(context.TODO(), nil)
			if err != nil {
				l.log.Error("issue getting L2 head", "err", err)
				continue
			}
			l.log.Warn("Got new L2 Block", "block", head.Number())
			if head.NumberU64() <= l.l2HeadNumber {
				// Didn't advance
				l.log.Trace("Old block")
				continue
			}
			if ch, err := derive.NewChannelOut(); err != nil {
				l.log.Error("Error creating channel", "err", err)
				continue
			} else {
				l.ch = ch
			}
			for i := l.l2HeadNumber + 1; i <= head.NumberU64(); i++ {
				block, err := l.cfg.L2Client.BlockByNumber(context.TODO(), new(big.Int).SetUint64(i))
				if err != nil {
					l.log.Error("issue getting L2 block", "err", err)
					continue
				}
				if err := l.ch.AddBlock(block); err != nil {
					l.log.Error("issue getting adding L2 Block", "err", err)
					continue
				}
				l.log.Warn("added L2 block to channel", "block_number", block.NumberU64(), "channel_id", l.ch.ID())
			}
			l.l2HeadNumber = head.NumberU64()

			if err := l.ch.Close(); err != nil {
				l.log.Error("issue getting adding L2 Block", "err", err)
				continue
			}
			data := new(bytes.Buffer)
			l.ch.OutputFrame(data, l.cfg.MaxL1TxSize)

			// Poll for new L1 Block and make a decision about what to do with the data

			//
			// Actually send the transaction
			//
			walletAddr := crypto.PubkeyToAddress(l.cfg.PrivKey.PublicKey)

			// Query for the submitter's current nonce.
			nonce, err := l.cfg.L1Client.NonceAt(ctx, walletAddr, nil)
			if err != nil {
				l.log.Error("unable to get current nonce", "err", err)
				continue
			}

			tx, err := l.CraftTx(ctx, data.Bytes(), nonce)
			if err != nil {
				l.log.Error("unable to craft tx", "err", err)
				continue
			}

			// Construct the a closure that will update the txn with the current gas prices.
			updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
				l.log.Debug("updating batch tx gas price")
				return l.UpdateGasPrice(ctx, tx)
			}

			// Wait until one of our submitted transactions confirms. If no
			// receipt is received it's likely our gas price was too low.
			receipt, err := l.txMgr.Send(ctx, updateGasPrice, l.cfg.L1Client.SendTransaction)
			if err != nil {
				l.log.Error("unable to publish tx", "err", err)
				continue
			}

			// The transaction was successfully submitted.
			l.log.Info("tx successfully published", "tx_hash", receipt.TxHash)

		case <-l.done:
			return
		}
	}
}

// NOTE: This method SHOULD NOT publish the resulting transaction.
func (l *BatchSubmitter) CraftTx(ctx context.Context, data []byte, nonce uint64) (*types.Transaction, error) {
	gasTipCap, err := l.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	head, err := l.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   l.cfg.ChainID,
		Nonce:     nonce,
		To:        &l.cfg.BatchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      data,
	}

	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true)
	if err != nil {
		return nil, err
	}
	rawTx.Gas = gas

	return types.SignNewTx(l.cfg.PrivKey, types.LatestSignerForChainID(l.cfg.ChainID), rawTx)
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: Thie method SHOULD NOT publish the resulting transaction.
func (l *BatchSubmitter) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	gasTipCap, err := l.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	head, err := l.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   l.cfg.ChainID,
		Nonce:     tx.Nonce(),
		To:        tx.To(),
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       tx.Gas(),
		Data:      tx.Data(),
	}

	return types.SignNewTx(l.cfg.PrivKey, types.LatestSignerForChainID(l.cfg.ChainID), rawTx)
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (l *BatchSubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return l.cfg.L1Client.SendTransaction(ctx, tx)
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

// parseAddress parses an ETH address from a hex string. This method will fail if
// the address is not a valid hexadecimal address.
func parseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}
