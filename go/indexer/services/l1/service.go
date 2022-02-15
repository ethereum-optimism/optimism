package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/server"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l1/bridge"

	_ "github.com/lib/pq"

	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
)

var logger = log.New("service", "l1")

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

// errWrongChainID represents the error when the configured chain id is not
// correct
var errWrongChainID = errors.New("wrong chain id provided")

var errNoNewBlocks = errors.New("no new blocks")

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
	RawL1Client             *rpc.Client
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

	bridges        []bridge.Bridge
	headerSelector *ConfirmedHeaderSelector

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64           `json:"synced"`
	Highest db.L1BlockLocator `json:"highest_block"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	bridges, err := bridge.BridgesByChainID(cfg.ChainID, cfg.L1Client, ctx)
	if err != nil {
		return nil, err
	}

	// Handle restart logic

	logger.Info("Creating L1 Indexer")

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
		cfg:            cfg,
		ctx:            ctx,
		cancel:         cancel,
		bridges:        bridges,
		headerSelector: confirmedHeaderSelector,
	}, nil
}

func (s *Service) Loop(ctx context.Context) {
	if err := s.catchUp(ctx); err != nil {
		if err == context.Canceled {
			return
		}

		logger.Error("error catching up to tip, trying to subscribe anyway", "err", err)
	}

	newHeads := make(chan *types.Header, 1000)
	go s.subscribeNewHeads(ctx, newHeads)

	for {
		select {
		case header := <-newHeads:
			logger.Info("Received new header", "header", header.Hash)
			for {
				err := s.Update(header)
				if err != nil && err != errNoNewBlocks {
					logger.Error("Unable to update indexer ", "err", err)
				}
				break
			}
		case <-s.ctx.Done():
			return
		}
	}
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

	headers := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.cfg.RawL1Client)
	if len(headers) == 0 {
		return errNoNewBlocks
	}

	if lowest.Number+1 != headers[0].Number.Uint64() {
		logger.Error("Block number does not immediately follow ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash,
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	if lowest.Hash != headers[0].ParentHash {
		logger.Error("Parent hash does not connect to ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash,
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()
	depositsByBlockhash := make(map[common.Hash][]db.Deposit)

	for _, bridge := range s.bridges {
		bridgeDeposits, err := bridge.GetDepositsByBlockRange(startHeight, endHeight)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		for blockHash, deposits := range bridgeDeposits {
			depositsByBlockhash[blockHash] = append(depositsByBlockhash[blockHash], deposits...)
		}
	}

	for _, header := range headers {
		blockHash := header.Hash
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
			logger.Error("Unable to import ",
				"block", number, "hash", blockHash, "err", err, "block", block)
			return err
		}

		logger.Debug("Imported ",
			"block", number, "hash", blockHash, "deposits", len(block.Deposits))
		for _, deposit := range block.Deposits {
			logger.Info("Indexed deposit ", "tx_hash", deposit.TxHash)
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
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	latestHeader, err := s.cfg.L1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	synced := float64(highestBlock.Number) / float64(latestHeader.Number.Int64())

	status := &IndexerStatus{
		Synced:  synced,
		Highest: *highestBlock,
	}

	server.RespondWithJSON(w, http.StatusOK, status)
}

func (s *Service) GetDeposits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil && limitStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if limit == 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil && offsetStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page := db.PaginationParam{
		Limit:  uint64(limit),
		Offset: uint64(offset),
	}

	deposits, err := s.cfg.DB.GetDepositsByAddress(common.HexToAddress(vars["address"]), page)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondWithJSON(w, http.StatusOK, deposits)
}

func (s *Service) subscribeNewHeads(ctx context.Context, heads chan *types.Header) {
	tick := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-tick.C:
			header, err := s.cfg.L1Client.HeaderByNumber(ctx, nil)
			if err != nil {
				logger.Error("error fetching header by number", "err", err)
			}
			heads <- header
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) catchUp(ctx context.Context) error {
	realHead, err := s.cfg.L1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return err
	}

	currHead, err := s.cfg.DB.GetHighestL2Block()
	if err != nil {
		return err
	}

	realHeadNum := realHead.Number.Uint64()
	var currHeadNum uint64
	if currHead != nil {
		currHeadNum = currHead.Number
	}

	if realHeadNum-s.cfg.ConfDepth <= currHeadNum+s.cfg.MaxHeaderBatchSize {
		return nil
	}

	logger.Info("chain is far behind head, resyncing")

	for realHeadNum-s.cfg.ConfDepth > currHeadNum+s.cfg.MaxHeaderBatchSize {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			if err := s.Update(realHead); err != nil {
				return err
			}
			currHead, err := s.cfg.DB.GetHighestL1Block()
			if err != nil {
				return err
			}
			currHeadNum = currHead.Number
		}
	}

	logger.Info("indexer is close enough to tip, starting regular loop")
	return nil
}

func (s *Service) Start() error {
	if s.cfg.ChainID == nil {
		return errNoChainID
	}
	s.wg.Add(1)
	go s.Loop(s.ctx)
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
