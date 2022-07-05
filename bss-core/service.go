package bsscore

import (
	"bytes"
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/bss-core/metrics"
	"github.com/ethereum-optimism/optimism/bss-core/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// weiToEth is the conversion rate from wei to ether.
	weiToEth = new(big.Float).SetFloat64(1e-18)
)

// Driver is an interface for creating and submitting batch transactions for a
// specific contract.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string

	// WalletAddr is the wallet address used to pay for batch transaction
	// fees.
	WalletAddr() common.Address

	// Metrics returns the subservice telemetry object.
	Metrics() metrics.Metrics

	// ClearPendingTx a publishes a transaction at the next available nonce in
	// order to clear any transactions in the mempool left over from a prior
	// running instance of the batch submitter.
	ClearPendingTx(context.Context, txmgr.TxManager, *ethclient.Client) error

	// GetBatchBlockRange returns the start and end L2 block heights that
	// need to be processed. Note that the end value is *exclusive*,
	// therefore if the returned values are identical nothing needs to be
	// processed.
	GetBatchBlockRange(ctx context.Context) (*big.Int, *big.Int, error)

	// CraftBatchTx transforms the L2 blocks between start and end into a batch
	// transaction using the given nonce. A dummy gas price is used in the
	// resulting transaction to use for size estimation. The driver may return a
	// nil value for transaction if there is no action that needs to be
	// performed.
	//
	// NOTE: This method SHOULD NOT publish the resulting transaction.
	CraftBatchTx(
		ctx context.Context,
		start, end, nonce *big.Int,
	) (*types.Transaction, error)

	// UpdateGasPrice signs an otherwise identical txn to the one provided but
	// with updated gas prices sampled from the existing network conditions.
	//
	// NOTE: Thie method SHOULD NOT publish the resulting transaction.
	UpdateGasPrice(
		ctx context.Context,
		tx *types.Transaction,
	) (*types.Transaction, error)

	// SendTransaction injects a signed transaction into the pending pool for
	// execution.
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

type ServiceConfig struct {
	Context         context.Context
	Driver          Driver
	PollInterval    time.Duration
	ClearPendingTx  bool
	L1Client        *ethclient.Client
	TxManagerConfig txmgr.Config
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	txMgr   txmgr.TxManager
	metrics metrics.Metrics

	wg sync.WaitGroup
}

func NewService(cfg ServiceConfig) *Service {
	ctx, cancel := context.WithCancel(cfg.Context)

	txMgr := txmgr.NewSimpleTxManager(
		cfg.Driver.Name(), cfg.TxManagerConfig, cfg.L1Client,
	)

	return &Service{
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		txMgr:   txMgr,
		metrics: cfg.Driver.Metrics(),
	}
}

func (s *Service) Start() error {
	s.wg.Add(1)
	go s.eventLoop()
	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Service) eventLoop() {
	defer s.wg.Done()

	name := s.cfg.Driver.Name()

	if s.cfg.ClearPendingTx {
		const maxClearRetries = 3
		for i := 0; i < maxClearRetries; i++ {
			err := s.cfg.Driver.ClearPendingTx(s.ctx, s.txMgr, s.cfg.L1Client)
			if err == nil {
				break
			} else if i < maxClearRetries-1 {
				continue
			}
			log.Crit("Unable to confirm a clearing transaction", "err", err)
		}
	}

	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Record the submitter's current ETH balance. This is done first in
			// case any of the remaining steps fail, we can at least have an
			// accurate view of the submitter's balance.
			balance, err := s.cfg.L1Client.BalanceAt(
				s.ctx, s.cfg.Driver.WalletAddr(), nil,
			)
			if err != nil {
				log.Error(name+" unable to get current balance", "err", err)
				continue
			}
			s.metrics.BalanceETH().Set(weiToEth64(balance))

			// Determine the range of L2 blocks that the batch submitter has not
			// processed, and needs to take action on.
			log.Info(name + " fetching current block range")
			start, end, err := s.cfg.Driver.GetBatchBlockRange(s.ctx)
			if err != nil {
				log.Error(name+" unable to get block range", "err", err)
				continue
			}

			// No new updates.
			if start.Cmp(end) == 0 {
				log.Info(name+" no updates", "start", start, "end", end)
				continue
			}
			log.Info(name+" block range", "start", start, "end", end)

			// Query for the submitter's current nonce.
			nonce64, err := s.cfg.L1Client.NonceAt(
				s.ctx, s.cfg.Driver.WalletAddr(), nil,
			)
			if err != nil {
				log.Error(name+" unable to get current nonce",
					"err", err)
				continue
			}
			nonce := new(big.Int).SetUint64(nonce64)

			batchTxBuildStart := time.Now()
			tx, err := s.cfg.Driver.CraftBatchTx(
				s.ctx, start, end, nonce,
			)
			if err != nil {
				log.Error(name+" unable to craft batch tx",
					"err", err)
				continue
			} else if tx == nil {
				continue
			}
			batchTxBuildTime := time.Since(batchTxBuildStart) / time.Millisecond
			s.metrics.BatchTxBuildTimeMs().Set(float64(batchTxBuildTime))

			// Record the size of the batch transaction.
			var txBuf bytes.Buffer
			if err := tx.EncodeRLP(&txBuf); err != nil {
				log.Error(name+" unable to encode batch tx", "err", err)
				continue
			}
			s.metrics.BatchSizeBytes().Observe(float64(len(txBuf.Bytes())))

			// Construct the transaction submission clousure that will attempt
			// to send the next transaction at the given nonce and gas price.
			updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
				log.Info(name+" updating batch tx gas price", "start", start,
					"end", end, "nonce", nonce)

				return s.cfg.Driver.UpdateGasPrice(ctx, tx)
			}

			// Wait until one of our submitted transactions confirms. If no
			// receipt is received it's likely our gas price was too low.
			batchConfirmationStart := time.Now()
			receipt, err := s.txMgr.Send(
				s.ctx, updateGasPrice, s.cfg.Driver.SendTransaction,
			)

			// Record the confirmation time and gas used if we receive a
			// receipt, as this indicates the transaction confirmed. We record
			// these metrics here as the transaction may have reverted, and will
			// abort below.
			if receipt != nil {
				batchConfirmationTime := time.Since(batchConfirmationStart) /
					time.Millisecond
				s.metrics.BatchConfirmationTimeMs().Set(float64(batchConfirmationTime))
				s.metrics.SubmissionGasUsedWei().Set(float64(receipt.GasUsed))
			}

			if err != nil {
				log.Error(name+" unable to publish batch tx",
					"err", err)
				s.metrics.FailedSubmissions().Inc()
				continue
			}

			// The transaction was successfully submitted.
			log.Info(name+" batch tx successfully published",
				"tx_hash", receipt.TxHash)
			s.metrics.BatchesSubmitted().Inc()
			s.metrics.SubmissionTimestamp().Set(float64(time.Now().UnixNano() / 1e6))

		case err := <-s.ctx.Done():
			log.Error(name+" service shutting down", "err", err)
			return
		}
	}
}

func weiToEth64(wei *big.Int) float64 {
	eth := new(big.Float).SetInt(wei)
	eth.Mul(eth, weiToEth)
	eth64, _ := eth.Float64()
	return eth64
}
