package indexer

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"sync"

	badger "github.com/dgraph-io/badger/v3"
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
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	metrics *metrics.Metrics

	db *badger.DB
	wg sync.WaitGroup
}

func NewService(cfg ServiceConfig) *Service {
	ctx, cancel := context.WithCancel(cfg.Context)

	db, err := badger.Open(badger.DefaultOptions("/tmp/optimism.indexer.db"))
	if err != nil {
		log.Println(err)
	}

	return &Service{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		db:     db,
	}
}

func (s *Service) Start() error {
	s.wg.Add(1)
	err := s.chainSync()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	s.wg.Wait()
	s.db.Close()
	return nil
}

func (s *Service) localHeight() (*big.Int, error) {
	var rawHeight []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("tip"))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			rawHeight = append([]byte{}, val...)
			return nil
		})
		return err
	})
	height := new(big.Int)
	height.SetBytes(rawHeight)
	return height, err
}

func (s *Service) chainSync() error {
	defer s.wg.Done()

	localHeight, err := s.localHeight()
	if err != nil && err != badger.ErrKeyNotFound {
		return err
	}
	tipHeader, err := s.cfg.L1Client.HeaderByNumber(s.cfg.Context, nil)
	if err != nil {
		return err
	}
	fmt.Printf("syncing to best height: %v; local height: %v\n", tipHeader.Number, localHeight)
	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("tip"), uint64ToBytes(tipHeader.Number.Uint64()))
	})
	return err
}
