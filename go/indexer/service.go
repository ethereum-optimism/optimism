package indexer

import (
	"context"
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// weiToGwei is the conversion rate from wei to gwei.
	weiToGwei = new(big.Float).SetFloat64(1e-18)
)

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Merge function to add two uint64 numbers
func add(existing, new []byte) []byte {
	return uint64ToBytes(bytesToUint64(existing) + bytesToUint64(new))
}

func weiToGwei64(wei *big.Int) float64 {
	gwei := new(big.Float).SetInt(wei)
	gwei.Mul(gwei, weiToGwei)
	gwei64, _ := gwei.Float64()
	return gwei64
}

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
	DB       *Database
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
	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	s.wg.Wait()
	if err := s.cfg.DB.Close(); err != nil {
		return err
	}
	return nil
}
