package l1

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
)

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

// errWrongChainID represents the error when the configured chain id is not
// correct
var errWrongChainID = errors.New("wrong chain id provided")

var errNoNewBlocks = errors.New("no new blocks")

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

type Backend interface {
	bind.ContractBackend
	HeaderBackend

	SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
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
	Context                 context.Context
	L1Client                *ethclient.Client
	ChainID                 *big.Int
	L1StandardBridgeAddress common.Address
	ConfDepth               uint64
	MaxHeaderBatchSize      uint64
	StartBlockNumber        uint64
	StartBlockHash          string
	DB                      *db.Database
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	l1BridgeContract *l1bridge.L1StandardBridgeFilterer
	backend          Backend
	headerSelector   HeaderSelector

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64           `json:"synced"`
	Highest db.L1BlockLocator `json:"highest_block"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	contract, err := l1bridge.NewL1StandardBridgeFilterer(cfg.L1StandardBridgeAddress, cfg.L1Client)
	if err != nil {
		return nil, err
	}

	// Handle restart logic

	log.Info("Creating L1 Indexer")

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

	confirmedHeaderSelector, err := NewConfirmedHeaderSelector(HeaderSelectorConfig{
		ConfDepth:    cfg.ConfDepth,
		MaxBatchSize: cfg.MaxHeaderBatchSize,
	})

	if err != nil {
		return nil, err
	}

	return &Service{
		cfg:              cfg,
		ctx:              ctx,
		cancel:           cancel,
		l1BridgeContract: contract,
		headerSelector:   confirmedHeaderSelector,
		backend:          cfg.L1Client,
	}, nil
}

func (s *Service) Loop(ctx context.Context) {
	newHeads := make(chan *types.Header, 1000)
	subscription, err := s.backend.SubscribeNewHead(s.ctx, newHeads)
	if err != nil {
		log.Error("unable to subscribe to new heads ", "err", err)
		s.Stop()
		return
	}
	defer subscription.Unsubscribe()

	for {
		select {
		case header := <-newHeads:
			log.Info("Received new header", "header", header.Hash)
			for {
				err := s.Update(header)
				if err != nil && err != errNoNewBlocks {
					log.Error("Unable to update indexer ", "err", err)
				}
				break
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) fetchETHBlockEventIterator(start, end uint64) (
	*l1bridge.L1StandardBridgeETHDepositInitiatedIterator, error) {

	const NUM_RETRIES = 5
	var err error
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(s.ctx, DefaultConnectionTimeout)

		var iter *l1bridge.L1StandardBridgeETHDepositInitiatedIterator
		iter, err = s.l1BridgeContract.FilterETHDepositInitiated(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil)
		if err != nil {
			log.Error("Unable to query deposit events for block range ",
				"start", start, "end", end, "error", err)
			cancel()
			continue
		}
		cancel()
		return iter, nil
	}
	return nil, err
}

func (s *Service) fetchERC20BlockEventIterator(start, end uint64) (
	*l1bridge.L1StandardBridgeERC20DepositInitiatedIterator, error) {

	const NUM_RETRIES = 5
	var err error
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(s.ctx, DefaultConnectionTimeout)

		var iter *l1bridge.L1StandardBridgeERC20DepositInitiatedIterator
		iter, err = s.l1BridgeContract.FilterERC20DepositInitiated(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil, nil)
		if err != nil {
			log.Error("Unable to query deposit events for block range ",
				"start", start, "end", end, "error", err)
			cancel()
			continue
		}
		cancel()
		return iter, nil
	}
	return nil, err
}

func (s *Service) Update(newHeader *types.Header) error {
	var lowest = db.L1BlockLocator{
		Number: s.cfg.StartBlockNumber,
		Hash:   common.HexToHash(s.cfg.StartBlockHash),
	}
	highestConfirmed, err := s.cfg.DB.GetHighestL1Block()
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
		log.Error("Block number does not immediately follow ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	if lowest.Hash != headers[0].ParentHash {
		log.Error("Parent hash does not connect to ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()
	depositsByBlockhash := make(map[common.Hash][]db.Deposit)

	ethIter, err := s.fetchETHBlockEventIterator(startHeight, endHeight)
	if err != nil {
		return err
	}

	for ethIter.Next() {
		depositsByBlockhash[ethIter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[ethIter.Event.Raw.BlockHash], db.Deposit{
				TxHash:      ethIter.Event.Raw.TxHash,
				FromAddress: ethIter.Event.From,
				ToAddress:   ethIter.Event.To,
				Amount:      ethIter.Event.Amount,
				Data:        ethIter.Event.Data,
				LogIndex:    ethIter.Event.Raw.Index,
			})
	}
	if err := ethIter.Error(); err != nil {
		return err
	}

	erc20Iter, err := s.fetchERC20BlockEventIterator(startHeight, endHeight)
	if err != nil {
		return err
	}

	for erc20Iter.Next() {
		depositsByBlockhash[erc20Iter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[erc20Iter.Event.Raw.BlockHash], db.Deposit{
				TxHash:      erc20Iter.Event.Raw.TxHash,
				L1Token:     erc20Iter.Event.L1Token,
				L2Token:     erc20Iter.Event.L2Token,
				FromAddress: erc20Iter.Event.From,
				ToAddress:   erc20Iter.Event.To,
				Amount:      erc20Iter.Event.Amount,
				Data:        erc20Iter.Event.Data,
				LogIndex:    erc20Iter.Event.Raw.Index,
			})
	}
	if err := erc20Iter.Error(); err != nil {
		return err
	}

	for _, header := range headers {
		blockHash := header.Hash()
		number := header.Number.Uint64()
		deposits := depositsByBlockhash[blockHash]

		block := &db.IndexedL1Block{
			Hash:       blockHash,
			ParentHash: header.ParentHash,
			Number:     number,
			Timestamp:  header.Time,
			Deposits:   deposits,
		}

		err := s.cfg.DB.AddIndexedL1Block(block)
		if err != nil {
			log.Error("Unable to import ",
				"block", number, "hash", blockHash, "err", err, "block", block)
			return err
		}

		log.Info("Imported ",
			"block", number, "hash", blockHash, "deposits", len(block.Deposits))
		for _, deposit := range block.Deposits {
			log.Info("Indexed deposit ", "tx_hash", deposit.TxHash)
		}
	}

	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
	newHeaderNumber := newHeader.Number.Uint64()
	if latestHeaderNumber+s.cfg.ConfDepth-1 == newHeaderNumber {
		return errNoNewBlocks
	}
	return nil
}

func (s *Service) GetIndexerStatus(w http.ResponseWriter, r *http.Request) {
	highestBlock, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	latestHeader, err := s.cfg.L1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	synced := float64(highestBlock.Number) / float64(latestHeader.Number.Int64())

	status := &IndexerStatus{
		Synced:  synced,
		Highest: *highestBlock,
	}

	respondWithJSON(w, http.StatusOK, status)
}

func (s *Service) GetDeposits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil && limitStr != "" {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if limit == 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil && offsetStr != "" {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page := db.PaginationParam{
		Limit:  uint64(limit),
		Offset: uint64(offset),
	}

	deposits, err := s.cfg.DB.GetDepositsByAddress(common.HexToAddress(vars["address"]), page)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, deposits)
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
