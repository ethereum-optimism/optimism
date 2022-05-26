package op_proposer

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// Driver is an interface for creating and submitting transactions for a
// specific contract.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string

	// WalletAddr is the wallet address used to pay for transaction fees.
	WalletAddr() common.Address

	// GetBlockRange returns the start and end L2 block heights that need to be
	// processed. Note that the end value is *exclusive*, therefore if the
	// returned values are identical nothing needs to be processed.
	GetBlockRange(ctx context.Context) (*big.Int, *big.Int, error)

	// CraftTx transforms the L2 blocks between start and end into a transaction
	// using the given nonce.
	//
	// NOTE: This method SHOULD NOT publish the resulting transaction.
	CraftTx(
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
	Log             log.Logger
	Context         context.Context
	Driver          Driver
	PollInterval    time.Duration
	L1Client        *ethclient.Client
	TxManagerConfig txmgr.Config
}

type Service struct {
	cfg   ServiceConfig
	txMgr txmgr.TxManager
	l     log.Logger

	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func NewService(cfg ServiceConfig) *Service {
	txMgr := txmgr.NewSimpleTxManager(
		cfg.Driver.Name(), cfg.TxManagerConfig, cfg.L1Client,
	)

	ctx, cancel := context.WithCancel(cfg.Context)

	return &Service{
		cfg:    cfg,
		txMgr:  txMgr,
		l:      cfg.Log,
		ctx:    ctx,
		cancel: cancel,
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
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Determine the range of L2 blocks that the submitter has not
			// processed, and needs to take action on.
			s.l.Info(name + " fetching current block range")
			start, end, err := s.cfg.Driver.GetBlockRange(s.ctx)
			if err != nil {
				s.l.Error(name+" unable to get block range", "err", err)
				continue
			}

			// No new updates.
			if start.Cmp(end) == 0 {
				s.l.Info(name+" no updates", "start", start, "end", end)
				continue
			}
			s.l.Info(name+" block range", "start", start, "end", end)

			// Query for the submitter's current nonce.
			nonce64, err := s.cfg.L1Client.NonceAt(
				s.ctx, s.cfg.Driver.WalletAddr(), nil,
			)
			if err != nil {
				s.l.Error(name+" unable to get current nonce",
					"err", err)
				continue
			}
			nonce := new(big.Int).SetUint64(nonce64)

			tx, err := s.cfg.Driver.CraftTx(
				s.ctx, start, end, nonce,
			)
			if err != nil {
				s.l.Error(name+" unable to craft tx",
					"err", err)
				continue
			}

			// Construct the a closure that will update the txn with the current
			// gas prices.
			updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
				s.l.Info(name+" updating batch tx gas price", "start", start,
					"end", end, "nonce", nonce)

				return s.cfg.Driver.UpdateGasPrice(ctx, tx)
			}

			// Wait until one of our submitted transactions confirms. If no
			// receipt is received it's likely our gas price was too low.
			receipt, err := s.txMgr.Send(
				s.ctx, updateGasPrice, s.cfg.Driver.SendTransaction,
			)
			if err != nil {
				s.l.Error(name+" unable to publish tx", "err", err)
				continue
			}

			// The transaction was successfully submitted.
			s.l.Info(name+" tx successfully published",
				"tx_hash", receipt.TxHash)

		case <-s.ctx.Done():
			s.l.Info(name + " service shutting down")
			return
		}
	}
}
