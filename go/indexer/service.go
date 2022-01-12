package indexer

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

// errWrongChainID represents the error when the configured chain id is not
// correct
var errWrongChainID = errors.New("wrong chain id provided")

var errNoNewBlocks = errors.New("no new blocks")

type Backend interface {
	bind.ContractBackend
	HeaderBackend

	SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error)
}

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
	Context            context.Context
	L1Client           *ethclient.Client
	ChainID            *big.Int
	CTCAddr            common.Address
	ConfDepth          uint64
	MaxHeaderBatchSize uint64
	StartBlockNumber   uint64
	StartBlockHash     string
	DB                 *Database
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	contract       *bindings.CanonicalTransactionChainFilterer
	backend        Backend
	headerSelector HeaderSelector

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	address := cfg.CTCAddr
	contract, err := bindings.NewCanonicalTransactionChainFilterer(address, cfg.L1Client)
	if err != nil {
		return nil, err
	}

	// Handle restart logic

	log.Info("Creating CTC Indexer")

	chainID, err := cfg.L1Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.ChainID != nil {
		if cfg.ChainID.Cmp(chainID) != 0 {
			return nil, fmt.Errorf("%w: configured with %d and got %d",
				errWrongChainID, cfg.ChainID, chainID)
		}
	} else {
		cfg.ChainID = chainID
	}

	return &Service{
		cfg:      cfg,
		ctx:      ctx,
		cancel:   cancel,
		contract: contract,
		headerSelector: NewConfirmedHeaderSelector(HeaderSelectorConfig{
			ConfDepth:    cfg.ConfDepth,
			MaxBatchSize: cfg.MaxHeaderBatchSize,
		}),
		backend: cfg.L1Client,
	}, nil
}

func (s *Service) Loop(ctx context.Context) {
	newHeads := make(chan *types.Header, 1000)
	subscription, err := s.backend.SubscribeNewHead(s.ctx, newHeads)
	if err != nil {
		panic(fmt.Sprintf("unable to subscribe to new heads: %v", err))
	}
	defer subscription.Unsubscribe()

	start := uint64(0)
	for {
		select {
		case header := <-newHeads:
			log.Info("Received new header", "header", header.Hash)
			for {
				err := s.Update(start, header)
				if err != nil && err != errNoNewBlocks {
					fmt.Printf("Unable to update indexer: %v\n", err)
				}
				break
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) fetchBlockEventIterator(start, end uint64) (
	*bindings.CanonicalTransactionChainTransactionEnqueuedIterator, error) {

	const NUM_RETRIES = 5
	var err error
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(s.ctx, DefaultConnectionTimeout)
		defer cancel()

		var iter *bindings.CanonicalTransactionChainTransactionEnqueuedIterator
		iter, err = s.contract.FilterTransactionEnqueued(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil, nil)
		if err != nil {
			fmt.Printf("Unable to query events for block range start=%d, end=%d; error=%v\n",
				start, end, err)
			continue
		}
		return iter, nil
	}
	return nil, err
}

func (s *Service) Update(start uint64, newHeader *types.Header) error {
	var lowest = BlockLocator{
		Number: s.cfg.StartBlockNumber,
		Hash:   common.HexToHash(s.cfg.StartBlockHash),
	}
	highestConfirmed, err := s.cfg.DB.GetHighestBlock()
	if err != nil {
		return err
	}
	if highestConfirmed != nil {
		lowest = *highestConfirmed
	}

	headers := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.backend)
	if len(headers) == 0 {
		return errNoNewBlocks
	}

	if lowest.Number+1 != headers[0].Number.Uint64() {
		fmt.Printf("Block number of block=%d hash=%s does not "+
			"immediately follow lowest block=%d hash=%s\n",
			headers[0].Number.Uint64(), headers[0].Hash(),
			lowest.Number, lowest.Hash)
		return nil
	}

	if lowest.Hash != headers[0].ParentHash {
		fmt.Printf("Parent hash of block=%d hash=%s does not "+
			"connect to lowest block=%d hash=%s\n", headers[0].Number.Uint64(),
			headers[0].Hash(), lowest.Number, lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()

	iter, err := s.fetchBlockEventIterator(startHeight, endHeight)
	if err != nil {
		return err
	}

	depositsByBlockhash := make(map[common.Hash][]Deposit)
	for iter.Next() {
		depositsByBlockhash[iter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[iter.Event.Raw.BlockHash], Deposit{
				QueueIndex: iter.Event.QueueIndex.Uint64(),
				TxHash:     iter.Event.Raw.TxHash,
				L1TxOrigin: iter.Event.L1TxOrigin,
				Target:     iter.Event.Target,
				GasLimit:   iter.Event.GasLimit,
				Data:       iter.Event.Data,
			})
	}
	if err := iter.Error(); err != nil {
		return err
	}

	for _, header := range headers {
		blockHash := header.Hash()
		number := header.Number.Uint64()
		deposits := depositsByBlockhash[blockHash]

		block := &IndexedBlock{
			Hash:       blockHash,
			ParentHash: header.ParentHash,
			Number:     number,
			Timestamp:  header.Time,
			Deposits:   deposits,
		}

		err := s.cfg.DB.AddIndexedBlock(block)
		if err != nil {
			fmt.Printf("Unable to import block=%d hash=%s err=%v "+
				"block: %v\n", number, blockHash, err, block)
			return err
		}

		fmt.Printf("Import block=%d hash=%s with %d deposits\n",
			number, blockHash, len(block.Deposits))
		for _, deposit := range block.Deposits {
			fmt.Printf("Deposit: l1_tx_origin=%s target=%s "+
				"gas_limit=%d queue_index=%d\n", deposit.L1TxOrigin,
				deposit.Target, deposit.GasLimit, deposit.QueueIndex)
		}
	}

	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
	newHeaderNumber := newHeader.Number.Uint64()
	if latestHeaderNumber+s.cfg.ConfDepth-1 == newHeaderNumber {
		return errNoNewBlocks
	}
	return nil
}

func (s *Service) Start() error {
	if s.cfg.ChainID == nil {
		return errNoChainID
	}
	s.wg.Add(1)
	go s.Loop(context.Background())
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
