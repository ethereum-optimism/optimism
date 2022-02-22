package l1

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1erc20"
	"github.com/ethereum-optimism/optimism/go/indexer/server"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l1/bridge"

	_ "github.com/lib/pq"

	"github.com/ethereum-optimism/optimism/go/indexer/db"
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

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

var ZeroAddress common.Address

// HeaderByNumberWithRetry retries the given func until it succeeds, waiting
// for clientRetryInterval duration after every call.
func HeaderByNumberWithRetry(ctx context.Context,
	client *ethclient.Client) (*types.Header, error) {
	for {
		res, err := client.HeaderByNumber(ctx, nil)
		switch err {
		case nil:
			return res, err
		default:
			log.Error("Error fetching header", "err", err)
			break
		}
		time.Sleep(clientRetryInterval)
	}
}

// Driver is an interface for indexing deposits from l1.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string
}

type ServiceConfig struct {
	Context                 context.Context
	Metrics                 *metrics.Metrics
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

	bridges        map[string]bridge.Bridge
	latestHeader   uint64
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

	// Handle restart logic

	logger.Info("Creating L1 Indexer")

	chainID, err := cfg.L1Client.ChainID(context.Background())
	if err != nil {
		cancel()
		return nil, err
	}

	if cfg.ChainID != nil {
		if cfg.ChainID.Cmp(chainID) != 0 {
			cancel()
			return nil, fmt.Errorf("%w: configured with %d and got %d",
				errWrongChainID, cfg.ChainID, chainID)
		}
	} else {
		cfg.ChainID = chainID
	}

	bridges, err := bridge.BridgesByChainID(cfg.ChainID, cfg.L1Client, ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	logger.Info("Scanning bridges for deposits", "bridges", bridges)

	confirmedHeaderSelector, err := NewConfirmedHeaderSelector(HeaderSelectorConfig{
		ConfDepth:    cfg.ConfDepth,
		MaxBatchSize: cfg.MaxHeaderBatchSize,
	})

	if err != nil {
		cancel()
		return nil, err
	}

	return &Service{
		cfg:            cfg,
		ctx:            ctx,
		cancel:         cancel,
		bridges:        bridges,
		headerSelector: confirmedHeaderSelector,
		metrics:        cfg.Metrics,
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
			if header == nil {
				break
			}

			logger.Info("Received new header", "header", header.Hash)
			atomic.StoreUint64(&s.latestHeader, header.Number.Uint64())
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
	tokensByAddress := map[common.Address]*db.Token{
		ZeroAddress: db.ETHL1Token,
	}

	start := prometheus.NewTimer(s.metrics.UpdateDuration.WithLabelValues("l1"))
	defer func() {
		dur := start.ObserveDuration()
		logger.Info("updated index", "start_height", startHeight, "end_height", endHeight, "duration", dur)
	}()

	for _, bridgeImpl := range s.bridges {
		bridgeDeposits, err := bridgeImpl.GetDepositsByBlockRange(startHeight, endHeight)
		if err != nil {
			logger.Error("an error occurred getting deposits", "err", err.Error(), "bridge", bridgeImpl.String())
			continue
		}

		// ERC20 deposits l1_token needs to be indexed before they can be
		// inserted, because l1_token is a foreign key to the token metadata
		switch bridgeImpl.(type) {
		case *bridge.StandardBridge:
			// Index L1 ERC20 tokens
			for _, deposits := range bridgeDeposits {
				for _, deposit := range deposits {
					if tokensByAddress[deposit.L1Token] != nil {
						continue
					}

					token, err := s.cfg.DB.GetL1TokenByAddress(deposit.L1Token.String())
					if err != nil {
						return err
					}
					if token != nil {
						tokensByAddress[deposit.L1Token] = token
						continue
					}

					contract, err := l1erc20.NewL1ERC20(deposit.L1Token, s.cfg.L1Client)
					if err != nil {
						return err
					}

					token, err = QueryERC20(deposit.L1Token, contract)
					if err != nil {
						logger.Error("Error querying ERC20 token details",
							"l1_token", deposit.L1Token.String(), "err", err)
						token = &db.Token{
							Address: deposit.L1Token.String(),
						}
					}
					if err := s.cfg.DB.AddL1Token(deposit.L1Token.String(), token); err != nil {
						return err
					}
					tokensByAddress[deposit.L1Token] = token
				}
			}
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
			logger.Error(
				"Unable to import ",
				"block", number,
				"hash", blockHash, "err", err,
				"block", block,
			)
			return err
		}

		logger.Debug("Imported ",
			"block", number, "hash", blockHash, "deposits", len(block.Deposits))
		for _, deposit := range block.Deposits {
			token := tokensByAddress[deposit.L1Token]
			logger.Info(
				"indexed deposit",
				"tx_hash", deposit.TxHash,
				"symbol", token.Symbol,
				"amount", deposit.Amount,
			)
			s.metrics.RecordDeposit(deposit.L1Token)
		}
	}

	newHeaderNumber := newHeader.Number.Uint64()
	s.metrics.SetL1SyncHeight(endHeight)
	s.metrics.SetL1SyncPercent(endHeight, newHeaderNumber)
	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
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

	var synced float64
	if s.latestHeader != 0 {
		synced = float64(highestBlock.Number) / float64(s.latestHeader)
	}

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
			header, err := HeaderByNumberWithRetry(ctx, s.cfg.L1Client)
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
	realHead, err := HeaderByNumberWithRetry(ctx, s.cfg.L1Client)
	if err != nil {
		return err
	}
	realHeadNum := realHead.Number.Uint64()

	currHead, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		return err
	}
	var currHeadNum uint64
	if currHead != nil {
		currHeadNum = currHead.Number
	}

	if realHeadNum-s.cfg.ConfDepth <= currHeadNum+s.cfg.MaxHeaderBatchSize {
		return nil
	}

	logger.Info("chain is far behind head, resyncing")
	s.metrics.SetL1CatchingUp(true)

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

	s.metrics.SetL1CatchingUp(false)
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
