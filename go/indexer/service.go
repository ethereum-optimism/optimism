package indexer

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// weiToGwei is the conversion rate from wei to gwei.
	weiToGwei = new(big.Float).SetFloat64(1e-18)
)

// Driver is an interface for indexing deposits from l1.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string

	// Metrics returns the subservice telemetry object.
	Metrics() *metrics.Metrics
}

type ServiceConfig struct {
	Context  context.Context
	Driver   Driver
	L1Client *ethclient.Client
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
