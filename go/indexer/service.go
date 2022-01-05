package indexer

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum-optimism/optimism/go/indexer/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// weiToGwei is the conversion rate from wei to gwei.
	weiToGwei = new(big.Float).SetFloat64(1e-18)
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
	Metrics() *metrics.Metrics

	// GetBatchBlockRange returns the start and end L2 block heights that
	// need to be processed. Note that the end value is *exclusive*,
	// therefore if the returned values are identical nothing needs to be
	// processed.
	GetBatchBlockRange(ctx context.Context) (*big.Int, *big.Int, error)

	// SubmitBatchTx transforms the L2 blocks between start and end into a
	// batch transaction using the given nonce and gasPrice. The final
	// transaction is published and returned to the call.
	SubmitBatchTx(
		ctx context.Context,
		start, end, nonce, gasPrice *big.Int,
	) (*types.Transaction, error)
}

type ServiceConfig struct {
	Context         context.Context
	Driver          Driver
	PollInterval    time.Duration
	L1Client        *ethclient.Client
	TxManagerConfig txmgr.Config
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

func NewService(cfg ServiceConfig) *Service {
	ctx, cancel := context.WithCancel(cfg.Context)

	return &Service{
		cfg:    cfg,
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
}

func weiToGwei64(wei *big.Int) float64 {
	gwei := new(big.Float).SetInt(wei)
	gwei.Mul(gwei, weiToGwei)
	gwei64, _ := gwei.Float64()
	return gwei64
}
